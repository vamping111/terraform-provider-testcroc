package autoscaling

import ( // nosemgrep: aws-sdk-go-multiple-service-imports
	"bytes"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/experimental/nullable"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceGroupCreate,
		Read:   resourceGroupRead,
		Update: resourceGroupUpdate,
		Delete: resourceGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validation.StringLenBetween(0, 255),
			},

			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.StringLenBetween(0, 255-resource.UniqueIDSuffixLength),
			},

			"launch_configuration": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"launch_configuration", "launch_template", "mixed_instances_policy"},
			},

			"launch_template": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:          schema.TypeString,
							Optional:      true,
							Computed:      true,
							ConflictsWith: []string{"launch_template.0.name"},
							ValidateFunc:  verify.ValidLaunchTemplateID,
						},
						"name": {
							Type:          schema.TypeString,
							Optional:      true,
							Computed:      true,
							ConflictsWith: []string{"launch_template.0.id"},
							ValidateFunc:  verify.ValidLaunchTemplateName,
						},
						"version": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(1, 255),
						},
					},
				},
				ExactlyOneOf: []string{"launch_configuration", "launch_template", "mixed_instances_policy"},
			},

			"mixed_instances_policy": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instances_distribution": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Computed: true,
							// Ideally we'd want to detect drift detection,
							// but a DiffSuppressFunc here does not behave nicely
							// for detecting missing configuration blocks
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									// These fields are returned from calls to the API
									// even if not provided at input time and can be omitted in requests;
									// thus, to prevent non-empty plans, we set these
									// to Computed and remove Defaults
									"on_demand_allocation_strategy": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"on_demand_base_capacity": {
										Type:         schema.TypeInt,
										Optional:     true,
										Computed:     true,
										ValidateFunc: validation.IntAtLeast(0),
									},
									"on_demand_percentage_above_base_capacity": {
										Type:         schema.TypeInt,
										Optional:     true,
										Computed:     true,
										ValidateFunc: validation.IntBetween(0, 100),
									},
									"spot_allocation_strategy": {
										Type:     schema.TypeString,
										Optional: true,
										Computed: true,
									},
									"spot_instance_pools": {
										Type:         schema.TypeInt,
										Optional:     true,
										Computed:     true,
										ValidateFunc: validation.IntAtLeast(0),
									},
									"spot_max_price": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"launch_template": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"launch_template_specification": {
										Type:     schema.TypeList,
										Required: true,
										MinItems: 1,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"launch_template_id": {
													Type:     schema.TypeString,
													Optional: true,
													Computed: true,
												},
												"launch_template_name": {
													Type:     schema.TypeString,
													Optional: true,
													Computed: true,
												},
												"version": {
													Type:     schema.TypeString,
													Optional: true,
													Default:  "$Default",
												},
											},
										},
									},
									"override": {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_type": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"launch_template_specification": {
													Type:     schema.TypeList,
													Optional: true,
													MinItems: 0,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"launch_template_id": {
																Type:     schema.TypeString,
																Optional: true,
																Computed: true,
															},
															"launch_template_name": {
																Type:     schema.TypeString,
																Optional: true,
																Computed: true,
															},
															"version": {
																Type:     schema.TypeString,
																Optional: true,
																Default:  "$Default",
															},
														},
													},
												},
												"weighted_capacity": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[1-9][0-9]{0,2}$`), "see https://docs.aws.amazon.com/autoscaling/ec2/APIReference/API_LaunchTemplateOverrides.html"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
				ExactlyOneOf: []string{"launch_configuration", "launch_template", "mixed_instances_policy"},
			},

			"capacity_rebalance": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"desired_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"min_elb_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"min_size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"max_size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"max_instance_lifetime": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"default_cooldown": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"force_delete": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"health_check_grace_period": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
			},

			"health_check_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"availability_zones": {
				Type:          schema.TypeSet,
				Optional:      true,
				Computed:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"vpc_zone_identifier"},
			},

			"placement_group": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"load_balancers": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_zone_identifier": {
				Type:          schema.TypeSet,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"availability_zones"},
				Elem:          &schema.Schema{Type: schema.TypeString},
				Set:           schema.HashString,
			},

			"termination_policies": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"wait_for_capacity_timeout": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "10m",
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					duration, err := time.ParseDuration(value)
					if err != nil {
						errors = append(errors, fmt.Errorf(
							"%q cannot be parsed as a duration: %s", k, err))
					}
					if duration < 0 {
						errors = append(errors, fmt.Errorf(
							"%q must be greater than zero", k))
					}
					return
				},
			},

			"wait_for_elb_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"enabled_metrics": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"suspended_processes": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"metrics_granularity": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "1Minute",
			},

			"protect_from_scale_in": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"target_group_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"initial_lifecycle_hook": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"default_result": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"heartbeat_timeout": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"lifecycle_transition": {
							Type:     schema.TypeString,
							Required: true,
						},
						"notification_metadata": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"notification_target_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"tag": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"tags"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},

						"value": {
							Type:     schema.TypeString,
							Required: true,
						},

						"propagate_at_launch": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				// This should be removable, but wait until other tags work is being done.
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["key"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["value"].(string)))
					buf.WriteString(fmt.Sprintf("%t-", m["propagate_at_launch"].(bool)))

					return create.StringHashcode(buf.String())
				},
			},

			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{Type: schema.TypeString},
				},
				ConflictsWith: []string{"tag"},
				Deprecated:    "Use tag instead",
				// Terraform 0.11 and earlier can provide incorrect type
				// information during difference handling, in which boolean
				// values are represented as "0" and "1". This Set function
				// normalizes these hashing variations, while the Terraform
				// Plugin SDK automatically suppresses the boolean/string
				// difference in the value itself.
				Set: func(v interface{}) int {
					var buf bytes.Buffer

					m, ok := v.(map[string]interface{})

					if !ok {
						return 0
					}

					if v, ok := m["key"].(string); ok {
						buf.WriteString(fmt.Sprintf("%s-", v))
					}

					if v, ok := m["value"].(string); ok {
						buf.WriteString(fmt.Sprintf("%s-", v))
					}

					if v, ok := m["propagate_at_launch"].(bool); ok {
						buf.WriteString(fmt.Sprintf("%t-", v))
					} else if v, ok := m["propagate_at_launch"].(string); ok {
						if b, err := strconv.ParseBool(v); err == nil {
							buf.WriteString(fmt.Sprintf("%t-", b))
						} else {
							buf.WriteString(fmt.Sprintf("%s-", v))
						}
					}

					return create.StringHashcode(buf.String())
				},
			},

			"service_linked_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"instance_refresh": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"strategy": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(autoscaling.RefreshStrategy_Values(), false),
						},
						"preferences": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"checkpoint_delay": {
										Type:         nullable.TypeNullableInt,
										Optional:     true,
										ValidateFunc: nullable.ValidateTypeStringNullableIntAtLeast(0),
									},
									"checkpoint_percentages": {
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeInt,
										},
									},
									"instance_warmup": {
										Type:         nullable.TypeNullableInt,
										Optional:     true,
										ValidateFunc: nullable.ValidateTypeStringNullableIntAtLeast(0),
									},
									"min_healthy_percentage": {
										Type:         schema.TypeInt,
										Optional:     true,
										Default:      90,
										ValidateFunc: validation.IntBetween(0, 100),
									},
								},
							},
						},
						"triggers": {
							Type:     schema.TypeSet,
							Optional: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateGroupInstanceRefreshTriggerFields,
							},
						},
					},
				},
			},

			"warm_pool": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"pool_state": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      "Stopped",
							ValidateFunc: validation.StringInSlice(autoscaling.WarmPoolState_Values(), false),
						},
						"min_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
						},
						"max_group_prepared_capacity": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"instance_reuse_policy": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"reuse_on_scale_in": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
								},
							},
						},
					},
				},
			},

			"force_delete_warm_pool": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},

		CustomizeDiff: customdiff.Sequence(
			customdiff.ComputedIf("launch_template.0.id", func(_ context.Context, diff *schema.ResourceDiff, meta interface{}) bool {
				return diff.HasChange("launch_template.0.name")
			}),
			customdiff.ComputedIf("launch_template.0.name", func(_ context.Context, diff *schema.ResourceDiff, meta interface{}) bool {
				return diff.HasChange("launch_template.0.id")
			}),
		),
	}
}

func generatePutLifecycleHookInputs(asgName string, cfgs []interface{}) []autoscaling.PutLifecycleHookInput {
	res := make([]autoscaling.PutLifecycleHookInput, 0, len(cfgs))

	for _, raw := range cfgs {
		cfg := raw.(map[string]interface{})

		input := autoscaling.PutLifecycleHookInput{
			AutoScalingGroupName: &asgName,
			LifecycleHookName:    aws.String(cfg["name"].(string)),
		}

		if v, ok := cfg["default_result"]; ok && v.(string) != "" {
			input.DefaultResult = aws.String(v.(string))
		}

		if v, ok := cfg["heartbeat_timeout"]; ok && v.(int) > 0 {
			input.HeartbeatTimeout = aws.Int64(int64(v.(int)))
		}

		if v, ok := cfg["lifecycle_transition"]; ok && v.(string) != "" {
			input.LifecycleTransition = aws.String(v.(string))
		}

		if v, ok := cfg["notification_metadata"]; ok && v.(string) != "" {
			input.NotificationMetadata = aws.String(v.(string))
		}

		if v, ok := cfg["notification_target_arn"]; ok && v.(string) != "" {
			input.NotificationTargetARN = aws.String(v.(string))
		}

		if v, ok := cfg["role_arn"]; ok && v.(string) != "" {
			input.RoleARN = aws.String(v.(string))
		}

		res = append(res, input)
	}

	return res
}

func resourceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn

	asgName := create.Name(d.Get("name").(string), d.Get("name_prefix").(string))

	createOpts := autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName:             aws.String(asgName),
		MixedInstancesPolicy:             expandMixedInstancesPolicy(d.Get("mixed_instances_policy").([]interface{})),
		NewInstancesProtectedFromScaleIn: aws.Bool(d.Get("protect_from_scale_in").(bool)),
	}
	updateOpts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgName),
	}

	initialLifecycleHooks := d.Get("initial_lifecycle_hook").(*schema.Set).List()
	twoPhases := len(initialLifecycleHooks) > 0

	minSize := aws.Int64(int64(d.Get("min_size").(int)))
	maxSize := aws.Int64(int64(d.Get("max_size").(int)))

	if twoPhases {
		createOpts.MinSize = aws.Int64(0)
		createOpts.MaxSize = aws.Int64(0)

		updateOpts.MinSize = minSize
		updateOpts.MaxSize = maxSize

		if v, ok := d.GetOk("desired_capacity"); ok {
			updateOpts.DesiredCapacity = aws.Int64(int64(v.(int)))
		}
	} else {
		createOpts.MinSize = minSize
		createOpts.MaxSize = maxSize

		if v, ok := d.GetOk("desired_capacity"); ok {
			createOpts.DesiredCapacity = aws.Int64(int64(v.(int)))
		}
	}

	if v, ok := d.GetOk("launch_configuration"); ok {
		createOpts.LaunchConfigurationName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("launch_template"); ok {
		createOpts.LaunchTemplate = expandLaunchTemplateSpecification(v.([]interface{}))
	}

	// Availability Zones are optional if VPC Zone Identifier(s) are specified
	if v, ok := d.GetOk("availability_zones"); ok && v.(*schema.Set).Len() > 0 {
		createOpts.AvailabilityZones = flex.ExpandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("tag"); ok {
		createOpts.Tags = Tags(KeyValueTags(v, asgName, TagResourceTypeGroup).IgnoreAWS())
	}

	if v, ok := d.GetOk("tags"); ok {
		createOpts.Tags = Tags(KeyValueTags(v, asgName, TagResourceTypeGroup).IgnoreAWS())
	}

	if v, ok := d.GetOk("capacity_rebalance"); ok {
		createOpts.CapacityRebalance = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("default_cooldown"); ok {
		createOpts.DefaultCooldown = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("health_check_type"); ok {
		createOpts.HealthCheckType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("health_check_grace_period"); ok {
		createOpts.HealthCheckGracePeriod = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("placement_group"); ok {
		createOpts.PlacementGroup = aws.String(v.(string))
	}

	if v, ok := d.GetOk("load_balancers"); ok && v.(*schema.Set).Len() > 0 {
		createOpts.LoadBalancerNames = flex.ExpandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("vpc_zone_identifier"); ok && v.(*schema.Set).Len() > 0 {
		createOpts.VPCZoneIdentifier = expandVpcZoneIdentifiers(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("termination_policies"); ok && len(v.([]interface{})) > 0 {
		createOpts.TerminationPolicies = flex.ExpandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("target_group_arns"); ok && len(v.(*schema.Set).List()) > 0 {
		createOpts.TargetGroupARNs = flex.ExpandStringSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk("service_linked_role_arn"); ok {
		createOpts.ServiceLinkedRoleARN = aws.String(v.(string))
	}

	if v, ok := d.GetOk("max_instance_lifetime"); ok {
		createOpts.MaxInstanceLifetime = aws.Int64(int64(v.(int)))
	}

	log.Printf("[DEBUG] Auto Scaling Group create configuration: %#v", createOpts)

	// Retry for IAM eventual consistency
	err := resource.Retry(propagationTimeout, func() *resource.RetryError {
		_, err := conn.CreateAutoScalingGroup(&createOpts)

		// ValidationError: You must use a valid fully-formed launch template. Value (tf-acc-test-6643732652421074386) for parameter iamInstanceProfile.name is invalid. Invalid IAM Instance Profile name
		if tfawserr.ErrMessageContains(err, "ValidationError", "Invalid IAM Instance Profile") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if tfresource.TimedOut(err) {
		_, err = conn.CreateAutoScalingGroup(&createOpts)
	}
	if err != nil {
		return fmt.Errorf("Error creating Auto Scaling Group: %s", err)
	}

	d.SetId(asgName)
	log.Printf("[INFO] Auto Scaling Group ID: %s", d.Id())

	if twoPhases {
		for _, hook := range generatePutLifecycleHookInputs(asgName, initialLifecycleHooks) {
			if err = resourceLifecycleHookPutOp(conn, &hook); err != nil {
				return fmt.Errorf("Error creating initial lifecycle hooks: %s", err)
			}
		}

		_, err = conn.UpdateAutoScalingGroup(&updateOpts)
		if err != nil {
			return fmt.Errorf("Error setting Auto Scaling Group initial capacity: %s", err)
		}
	}

	if err := waitForASGCapacity(d, meta, CapacitySatisfiedCreate); err != nil {
		return err
	}

	if _, ok := d.GetOk("suspended_processes"); ok {
		suspendedProcessesErr := enableASGSuspendedProcesses(d, conn)
		if suspendedProcessesErr != nil {
			return suspendedProcessesErr
		}
	}

	if _, ok := d.GetOk("enabled_metrics"); ok {
		metricsErr := enableASGMetricsCollection(d, conn)
		if metricsErr != nil {
			return metricsErr
		}
	}

	if _, ok := d.GetOk("warm_pool"); ok {
		_, err := conn.PutWarmPool(CreatePutWarmPoolInput(d.Id(), d.Get("warm_pool").([]interface{})))

		if err != nil {
			return fmt.Errorf("error creating Warm Pool for Auto Scaling Group (%s), error: %s", d.Id(), err)
		}

		log.Printf("[INFO] Successfully created Warm pool")

	}

	return resourceGroupRead(d, meta)
}

// TODO: wrap all top-level error returns
func resourceGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	g, err := getGroup(d.Id(), conn)
	if err != nil {
		return err
	}
	if g == nil && !d.IsNewResource() {
		log.Printf("[WARN] Auto Scaling Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("availability_zones", flex.FlattenStringList(g.AvailabilityZones)); err != nil {
		return fmt.Errorf("error setting availability_zones: %s", err)
	}

	d.Set("arn", g.AutoScalingGroupARN)
	d.Set("capacity_rebalance", g.CapacityRebalance)
	d.Set("default_cooldown", g.DefaultCooldown)
	d.Set("desired_capacity", g.DesiredCapacity)

	d.Set("enabled_metrics", nil)
	d.Set("metrics_granularity", "1Minute")
	if g.EnabledMetrics != nil {
		if err := d.Set("enabled_metrics", flattenASGEnabledMetrics(g.EnabledMetrics)); err != nil {
			return fmt.Errorf("error setting enabled_metrics: %s", err)
		}
		d.Set("metrics_granularity", g.EnabledMetrics[0].Granularity)
	}

	d.Set("health_check_grace_period", g.HealthCheckGracePeriod)
	d.Set("health_check_type", g.HealthCheckType)

	if err := d.Set("load_balancers", flex.FlattenStringList(g.LoadBalancerNames)); err != nil {
		return fmt.Errorf("error setting load_balancers: %s", err)
	}

	d.Set("launch_configuration", g.LaunchConfigurationName)

	if err := d.Set("launch_template", flattenLaunchTemplateSpecificationMap(g.LaunchTemplate)); err != nil {
		return fmt.Errorf("error setting launch_template: %s", err)
	}

	d.Set("max_size", g.MaxSize)
	d.Set("min_size", g.MinSize)

	if err := d.Set("mixed_instances_policy", flattenMixedInstancesPolicy(g.MixedInstancesPolicy)); err != nil {
		return fmt.Errorf("error setting mixed_instances_policy: %s", err)
	}

	d.Set("name", g.AutoScalingGroupName)
	d.Set("name_prefix", create.NamePrefixFromName(aws.StringValue(g.AutoScalingGroupName)))
	d.Set("placement_group", g.PlacementGroup)
	d.Set("protect_from_scale_in", g.NewInstancesProtectedFromScaleIn)
	d.Set("service_linked_role_arn", g.ServiceLinkedRoleARN)
	d.Set("max_instance_lifetime", g.MaxInstanceLifetime)

	if err := d.Set("suspended_processes", flattenASGSuspendedProcesses(g.SuspendedProcesses)); err != nil {
		return fmt.Errorf("error setting suspended_processes: %s", err)
	}

	var tagOk, tagsOk bool
	var v interface{}

	// Deprecated: In a future major version, this should always set all tags except those ignored.
	//             Remove d.GetOk() and Only() handling.
	if v, tagOk = d.GetOk("tag"); tagOk {
		proposedStateTags := KeyValueTags(v, d.Id(), TagResourceTypeGroup)

		if err := d.Set("tag", ListOfMap(KeyValueTags(g.Tags, d.Id(), TagResourceTypeGroup).IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Only(proposedStateTags))); err != nil {
			return fmt.Errorf("error setting tag: %w", err)
		}
	}

	if v, tagsOk = d.GetOk("tags"); tagsOk {
		proposedStateTags := KeyValueTags(v, d.Id(), TagResourceTypeGroup)

		if err := d.Set("tags", ListOfStringMap(KeyValueTags(g.Tags, d.Id(), TagResourceTypeGroup).IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Only(proposedStateTags))); err != nil {
			return fmt.Errorf("error setting tags: %w", err)
		}
	}

	if !tagOk && !tagsOk {
		if err := d.Set("tag", ListOfMap(KeyValueTags(g.Tags, d.Id(), TagResourceTypeGroup).IgnoreAWS().IgnoreConfig(ignoreTagsConfig))); err != nil {
			return fmt.Errorf("error setting tag: %w", err)
		}
	}

	if err := d.Set("target_group_arns", flex.FlattenStringList(g.TargetGroupARNs)); err != nil {
		return fmt.Errorf("error setting target_group_arns: %s", err)
	}

	// If no termination polices are explicitly configured and the upstream state
	// is only using the "Default" policy, clear the state to make it consistent
	// with the default AWS create API behavior.
	_, ok := d.GetOk("termination_policies")
	if !ok && len(g.TerminationPolicies) == 1 && aws.StringValue(g.TerminationPolicies[0]) == "Default" {
		d.Set("termination_policies", []interface{}{})
	} else {
		if err := d.Set("termination_policies", flex.FlattenStringList(g.TerminationPolicies)); err != nil {
			return fmt.Errorf("error setting termination_policies: %s", err)
		}
	}

	d.Set("vpc_zone_identifier", []string{})
	if len(aws.StringValue(g.VPCZoneIdentifier)) > 0 {
		if err := d.Set("vpc_zone_identifier", strings.Split(aws.StringValue(g.VPCZoneIdentifier), ",")); err != nil {
			return fmt.Errorf("error setting vpc_zone_identifier: %s", err)
		}
	}

	if err := d.Set("warm_pool", FlattenWarmPoolConfiguration(g.WarmPoolConfiguration)); err != nil {
		return fmt.Errorf("error setting warm_pool for Auto Scaling Group (%s), error: %s", d.Id(), err)
	}

	return nil
}

func waitUntilGroupLoadBalancerTargetGroupsRemoved(conn *autoscaling.AutoScaling, asgName string) error {
	input := &autoscaling.DescribeLoadBalancerTargetGroupsInput{
		AutoScalingGroupName: aws.String(asgName),
	}
	var tgRemoving bool

	for {
		output, err := conn.DescribeLoadBalancerTargetGroups(input)

		if err != nil {
			return err
		}

		for _, tg := range output.LoadBalancerTargetGroups {
			if aws.StringValue(tg.State) == "Removing" {
				tgRemoving = true
				break
			}
		}

		if tgRemoving {
			tgRemoving = false
			input.NextToken = nil
			continue
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil
}

func waitUntilGroupLoadBalancerTargetGroupsAdded(conn *autoscaling.AutoScaling, asgName string) error {
	input := &autoscaling.DescribeLoadBalancerTargetGroupsInput{
		AutoScalingGroupName: aws.String(asgName),
	}
	var tgAdding bool

	for {
		output, err := conn.DescribeLoadBalancerTargetGroups(input)

		if err != nil {
			return err
		}

		for _, tg := range output.LoadBalancerTargetGroups {
			if aws.StringValue(tg.State) == "Adding" {
				tgAdding = true
				break
			}
		}

		if tgAdding {
			tgAdding = false
			input.NextToken = nil
			continue
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil
}

func resourceGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn
	shouldWaitForCapacity := false
	shouldRefreshInstances := false

	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
	}

	opts.NewInstancesProtectedFromScaleIn = aws.Bool(d.Get("protect_from_scale_in").(bool))

	if d.HasChange("default_cooldown") {
		opts.DefaultCooldown = aws.Int64(int64(d.Get("default_cooldown").(int)))
	}

	if d.HasChange("capacity_rebalance") {
		// If the capacity rebalance field is set to null, we need to explicitly set
		// it back to "false", or the API won't reset it for us.
		if v, ok := d.GetOk("capacity_rebalance"); ok {
			opts.CapacityRebalance = aws.Bool(v.(bool))
		} else {
			opts.CapacityRebalance = aws.Bool(false)
		}
	}

	if d.HasChange("desired_capacity") {
		opts.DesiredCapacity = aws.Int64(int64(d.Get("desired_capacity").(int)))
		shouldWaitForCapacity = true
	}

	if d.HasChange("launch_configuration") {
		if v, ok := d.GetOk("launch_configuration"); ok {
			opts.LaunchConfigurationName = aws.String(v.(string))
		}
		shouldRefreshInstances = true
	}

	if d.HasChange("launch_template") {
		if v, ok := d.GetOk("launch_template"); ok && len(v.([]interface{})) > 0 {
			opts.LaunchTemplate = expandLaunchTemplateSpecification(v.([]interface{}))
		}
		shouldRefreshInstances = true
	}

	if d.HasChange("mixed_instances_policy") {
		opts.MixedInstancesPolicy = expandMixedInstancesPolicy(d.Get("mixed_instances_policy").([]interface{}))
		shouldRefreshInstances = true
	}

	if d.HasChange("min_size") {
		opts.MinSize = aws.Int64(int64(d.Get("min_size").(int)))
		shouldWaitForCapacity = true
	}

	if d.HasChange("max_size") {
		opts.MaxSize = aws.Int64(int64(d.Get("max_size").(int)))
	}

	if d.HasChange("max_instance_lifetime") {
		opts.MaxInstanceLifetime = aws.Int64(int64(d.Get("max_instance_lifetime").(int)))
	}

	if d.HasChange("health_check_grace_period") {
		opts.HealthCheckGracePeriod = aws.Int64(int64(d.Get("health_check_grace_period").(int)))
	}

	if d.HasChange("health_check_type") {
		opts.HealthCheckGracePeriod = aws.Int64(int64(d.Get("health_check_grace_period").(int)))
		opts.HealthCheckType = aws.String(d.Get("health_check_type").(string))
	}

	if d.HasChange("vpc_zone_identifier") {
		opts.VPCZoneIdentifier = expandVpcZoneIdentifiers(d.Get("vpc_zone_identifier").(*schema.Set).List())
	}

	if d.HasChange("availability_zones") {
		if v, ok := d.GetOk("availability_zones"); ok && v.(*schema.Set).Len() > 0 {
			opts.AvailabilityZones = flex.ExpandStringSet(v.(*schema.Set))
		}
	}

	if d.HasChange("placement_group") {
		opts.PlacementGroup = aws.String(d.Get("placement_group").(string))
	}

	if d.HasChange("termination_policies") {
		// If the termination policy is set to null, we need to explicitly set
		// it back to "Default", or the API won't reset it for us.
		if v, ok := d.GetOk("termination_policies"); ok && len(v.([]interface{})) > 0 {
			opts.TerminationPolicies = flex.ExpandStringList(v.([]interface{}))
		} else {
			log.Printf("[DEBUG] Explicitly setting null termination policy to 'Default'")
			opts.TerminationPolicies = aws.StringSlice([]string{"Default"})
		}
	}

	if d.HasChange("service_linked_role_arn") {
		opts.ServiceLinkedRoleARN = aws.String(d.Get("service_linked_role_arn").(string))
	}

	if d.HasChanges("tag", "tags") {
		oTagRaw, nTagRaw := d.GetChange("tag")
		oTagsRaw, nTagsRaw := d.GetChange("tags")

		oTag := KeyValueTags(oTagRaw, d.Id(), TagResourceTypeGroup)
		oTags := KeyValueTags(oTagsRaw, d.Id(), TagResourceTypeGroup)
		oldTags := Tags(oTag.Merge(oTags))

		nTag := KeyValueTags(nTagRaw, d.Id(), TagResourceTypeGroup)
		nTags := KeyValueTags(nTagsRaw, d.Id(), TagResourceTypeGroup)
		newTags := Tags(nTag.Merge(nTags))

		if err := UpdateTags(conn, d.Id(), TagResourceTypeGroup, oldTags, newTags); err != nil {
			return fmt.Errorf("error updating tags for Auto Scaling Group (%s): %w", d.Id(), err)
		}
	}

	log.Printf("[DEBUG] Auto Scaling Group update configuration: %#v", opts)
	_, err := conn.UpdateAutoScalingGroup(&opts)
	if err != nil {
		return fmt.Errorf("Error updating Auto Scaling Group: %s", err)
	}

	if d.HasChange("load_balancers") {

		o, n := d.GetChange("load_balancers")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := flex.ExpandStringSet(os.Difference(ns))
		add := flex.ExpandStringSet(ns.Difference(os))

		if len(remove) > 0 {
			// API only supports removing 10 at a time
			var batches [][]*string

			batchSize := 10

			for batchSize < len(remove) {
				remove, batches = remove[batchSize:], append(batches, remove[0:batchSize:batchSize])
			}
			batches = append(batches, remove)

			for _, batch := range batches {
				_, err := conn.DetachLoadBalancers(&autoscaling.DetachLoadBalancersInput{
					AutoScalingGroupName: aws.String(d.Id()),
					LoadBalancerNames:    batch,
				})

				if err != nil {
					return fmt.Errorf("error detaching Auto Scaling Group (%s) Load Balancers: %s", d.Id(), err)
				}

				if err := waitUntilGroupLoadBalancersRemoved(conn, d.Id()); err != nil {
					return fmt.Errorf("error describing Auto Scaling Group (%s) Load Balancers being removed: %s", d.Id(), err)
				}
			}
		}

		if len(add) > 0 {
			// API only supports adding 10 at a time
			batchSize := 10

			var batches [][]*string

			for batchSize < len(add) {
				add, batches = add[batchSize:], append(batches, add[0:batchSize:batchSize])
			}
			batches = append(batches, add)

			for _, batch := range batches {
				_, err := conn.AttachLoadBalancers(&autoscaling.AttachLoadBalancersInput{
					AutoScalingGroupName: aws.String(d.Id()),
					LoadBalancerNames:    batch,
				})

				if err != nil {
					return fmt.Errorf("error attaching Auto Scaling Group (%s) Load Balancers: %s", d.Id(), err)
				}

				if err := waitUntilGroupLoadBalancersAdded(conn, d.Id()); err != nil {
					return fmt.Errorf("error describing Auto Scaling Group (%s) Load Balancers being added: %s", d.Id(), err)
				}
			}
		}
	}

	if d.HasChange("target_group_arns") {

		o, n := d.GetChange("target_group_arns")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := flex.ExpandStringSet(os.Difference(ns))
		add := flex.ExpandStringSet(ns.Difference(os))

		if len(remove) > 0 {
			// AWS API only supports adding/removing 10 at a time
			var batches [][]*string

			batchSize := 10

			for batchSize < len(remove) {
				remove, batches = remove[batchSize:], append(batches, remove[0:batchSize:batchSize])
			}
			batches = append(batches, remove)

			for _, batch := range batches {
				_, err := conn.DetachLoadBalancerTargetGroups(&autoscaling.DetachLoadBalancerTargetGroupsInput{
					AutoScalingGroupName: aws.String(d.Id()),
					TargetGroupARNs:      batch,
				})
				if err != nil {
					return fmt.Errorf("Error updating Load Balancers Target Groups for Auto Scaling Group (%s), error: %s", d.Id(), err)
				}

				if err := waitUntilGroupLoadBalancerTargetGroupsRemoved(conn, d.Id()); err != nil {
					return fmt.Errorf("error describing Auto Scaling Group (%s) Load Balancer Target Groups being removed: %s", d.Id(), err)
				}
			}

		}

		if len(add) > 0 {
			batchSize := 10

			var batches [][]*string

			for batchSize < len(add) {
				add, batches = add[batchSize:], append(batches, add[0:batchSize:batchSize])
			}
			batches = append(batches, add)

			for _, batch := range batches {
				_, err := conn.AttachLoadBalancerTargetGroups(&autoscaling.AttachLoadBalancerTargetGroupsInput{
					AutoScalingGroupName: aws.String(d.Id()),
					TargetGroupARNs:      batch,
				})

				if err != nil {
					return fmt.Errorf("Error updating Load Balancers Target Groups for Auto Scaling Group (%s), error: %s", d.Id(), err)
				}

				if err := waitUntilGroupLoadBalancerTargetGroupsAdded(conn, d.Id()); err != nil {
					return fmt.Errorf("error describing Auto Scaling Group (%s) Load Balancer Target Groups being added: %s", d.Id(), err)
				}
			}
		}
	}

	if instanceRefreshRaw, ok := d.GetOk("instance_refresh"); ok {
		instanceRefresh := instanceRefreshRaw.([]interface{})
		if !shouldRefreshInstances {
			if len(instanceRefresh) > 0 && instanceRefresh[0] != nil {
				m := instanceRefresh[0].(map[string]interface{})
				attrsSet := m["triggers"].(*schema.Set)
				attrs := attrsSet.List()
				strs := make([]string, len(attrs))
				for i, a := range attrs {
					strs[i] = a.(string)
				}
				if attrsSet.Contains("tag") && !attrsSet.Contains("tags") {
					strs = append(strs, "tags") // nozero
				} else if !attrsSet.Contains("tag") && attrsSet.Contains("tags") {
					strs = append(strs, "tag") // nozero
				}
				shouldRefreshInstances = d.HasChanges(strs...)
			}
		}
		if shouldRefreshInstances {
			if err := GroupRefreshInstances(conn, d.Id(), instanceRefresh); err != nil {
				return fmt.Errorf("failed to start instance refresh of Auto Scaling Group %s: %w", d.Id(), err)
			}
		}
	}

	if d.HasChange("warm_pool") {
		w := d.Get("warm_pool").([]interface{})

		// No warm pool exists in new config. Delete it.
		if len(w) == 0 || w[0] == nil {
			g, err := getGroup(d.Id(), conn)
			if err != nil {
				return err
			}

			if err := resourceGroupWarmPoolDelete(g, d, meta); err != nil {
				return fmt.Errorf("error deleting Warm pool for Auto Scaling Group %s: %s", d.Id(), err)
			}

			log.Printf("[INFO] Successfully removed Warm pool")
		} else {
			_, err := conn.PutWarmPool(CreatePutWarmPoolInput(d.Id(), d.Get("warm_pool").([]interface{})))

			if err != nil {
				return fmt.Errorf("error updating Warm Pool for Auto Scaling Group (%s), error: %s", d.Id(), err)
			}

			log.Printf("[INFO] Successfully updated Warm pool")
		}
	}

	if shouldWaitForCapacity {
		if err := waitForASGCapacity(d, meta, CapacitySatisfiedUpdate); err != nil {
			return fmt.Errorf("error waiting for Auto Scaling Group Capacity: %w", err)
		}
	}

	if d.HasChange("enabled_metrics") {
		if err := updateASGMetricsCollection(d, conn); err != nil {
			return fmt.Errorf("Error updating Auto Scaling Group Metrics collection: %s", err)
		}
	}

	if d.HasChange("suspended_processes") {
		if err := updateASGSuspendedProcesses(d, conn); err != nil {
			return fmt.Errorf("Error updating Auto Scaling Group Suspended Processes: %s", err)
		}
	}

	return resourceGroupRead(d, meta)
}

func resourceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn

	// Read the Auto Scaling Group first. If it doesn't exist, we're done.
	// We need the group in order to check if there are instances attached.
	// If so, we need to remove those first.
	g, err := getGroup(d.Id(), conn)
	if err != nil {
		return err
	}
	if g == nil {
		log.Printf("[WARN] Auto Scaling Group (%s) not found, removing from state", d.Id())
		return nil
	}

	// Try deleting Warm pool first.
	if err := resourceGroupWarmPoolDelete(g, d, meta); err != nil {
		return fmt.Errorf("error deleting Warm pool for Auto Scaling Group %s: %s", d.Id(), err)
	}

	if len(g.Instances) > 0 || aws.Int64Value(g.DesiredCapacity) > 0 {
		if err := resourceGroupDrain(d, meta); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] Auto Scaling Group destroy: %v", d.Id())
	deleteopts := autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
		ForceDelete:          aws.Bool(d.Get("force_delete").(bool)),
	}

	// We retry the delete operation to handle InUse/InProgress errors coming
	// from scaling operations. We should be able to sneak in a delete in between
	// scaling operations within 5m.
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		if _, err := conn.DeleteAutoScalingGroup(&deleteopts); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				switch awserr.Code() {
				case "InvalidGroup.NotFound":
					// Already gone? Sure!
					return nil
				case "ResourceInUse", "ScalingActivityInProgress":
					// These are retryable
					return resource.RetryableError(awserr)
				}
			}
			// Didn't recognize the error, so shouldn't retry.
			return resource.NonRetryableError(err)
		}
		// Successful delete
		return nil
	})
	if tfresource.TimedOut(err) {
		_, err = conn.DeleteAutoScalingGroup(&deleteopts)
		if tfawserr.ErrCodeEquals(err, "InvalidGroup.NotFound") {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("Error deleting Auto Scaling Group: %s", err)
	}

	var group *autoscaling.Group
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		group, err = getGroup(d.Id(), conn)

		if group != nil {
			return resource.RetryableError(fmt.Errorf("Auto Scaling Group still exists"))
		}
		return nil
	})
	if tfresource.TimedOut(err) {
		group, err = getGroup(d.Id(), conn)
		if group != nil {
			return fmt.Errorf("Auto Scaling Group still exists")
		}
	}
	if err != nil {
		return fmt.Errorf("Error deleting Auto Scaling Group: %s", err)
	}
	return nil
}

func resourceGroupWarmPoolDelete(g *autoscaling.Group, d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn

	if g.WarmPoolConfiguration == nil {
		// No warm pool configured. Skipping deletion.
		return nil
	}

	log.Printf("[INFO] Auto Scaling Group has a Warm Pool. First deleting it.")

	if err := resourceGroupWarmPoolDrain(d, meta); err != nil {
		return err
	}

	// Delete Warm Pool if it is not pending delete.
	if g.WarmPoolConfiguration.Status == nil || aws.StringValue(g.WarmPoolConfiguration.Status) != "PendingDelete" {
		deleteopts := autoscaling.DeleteWarmPoolInput{
			AutoScalingGroupName: aws.String(d.Id()),
			ForceDelete:          aws.Bool(d.Get("force_delete").(bool) || d.Get("force_delete_warm_pool").(bool)),
		}

		err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
			_, err := conn.DeleteWarmPool(&deleteopts)
			if err != nil {
				if callerr, ok := err.(awserr.Error); ok {
					switch callerr.Code() {
					case "ResourceInUse", "ScalingActivityInProgress":
						// These are retryable
						return resource.RetryableError(callerr)
					}
				}
				// Didn't recognize the error, so shouldn't retry.
				return resource.NonRetryableError(err)
			}
			// Successful delete
			return nil
		})

		if tfresource.TimedOut(err) {
			_, err = conn.DeleteWarmPool(&deleteopts)
		}
		if err != nil {
			return fmt.Errorf("error deleting Warm Pool: %s", err)
		}
	}

	// Wait for Warm pool to be gone.
	if err := waitForWarmPoolDeletion(conn, d); err != nil {
		return fmt.Errorf("error waiting for Warm Pool deletion: %s", err)
	}

	log.Printf("[INFO] Successfully removed Warm pool")

	return nil
}

func waitForWarmPoolDeletion(conn *autoscaling.AutoScaling, d *schema.ResourceData) error {
	stateConf := &resource.StateChangeConf{
		Pending:        []string{"", autoscaling.WarmPoolStatusPendingDelete},
		Target:         []string{"deleted"},
		Refresh:        asgWarmPoolStateRefreshFunc(conn, d.Id()),
		Timeout:        d.Timeout(schema.TimeoutDelete),
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for Auto Scaling Group (%s) Warm Pool deletion", d.Id())
	_, err := stateConf.WaitForState()

	if tfresource.NotFound(err) {
		return nil
	}

	return err
}

func asgWarmPoolStateRefreshFunc(conn *autoscaling.AutoScaling, asgName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		g, err := getGroup(asgName, conn)

		if err != nil {
			return nil, "", fmt.Errorf("error describing Auto Scaling Group (%s): %s", asgName, err)
		}

		if g == nil || g.WarmPoolConfiguration == nil {
			return nil, "deleted", nil
		}
		return asgName, aws.StringValue(g.WarmPoolConfiguration.Status), nil
	}
}

func getGroupWarmPool(asgName string, conn *autoscaling.AutoScaling) (*autoscaling.DescribeWarmPoolOutput, error) {
	describeOpts := autoscaling.DescribeWarmPoolInput{
		AutoScalingGroupName: aws.String(asgName),
	}

	log.Printf("[DEBUG] Warm Pool describe configuration input: %#v", describeOpts)
	describeWarmPoolOutput, err := conn.DescribeWarmPool(&describeOpts)
	if err != nil {
		autoscalingerr, ok := err.(awserr.Error)
		if ok && autoscalingerr.Code() == "InvalidGroup.NotFound" {
			return nil, nil
		}

		return nil, fmt.Errorf("error retrieving Warm Pool: %s", err)
	}

	return describeWarmPoolOutput, nil
}

func resourceGroupWarmPoolDrain(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn

	if d.Get("force_delete").(bool) || d.Get("force_delete_warm_pool").(bool) {
		log.Printf("[DEBUG] Skipping Warm pool drain, force delete was set.")
		return nil
	}

	// First, set the max group prepared capacity and min size to zero for the pool to drain
	log.Printf("[DEBUG] Reducing Warm pool capacity to zero")
	opts := autoscaling.PutWarmPoolInput{
		AutoScalingGroupName:     aws.String(d.Id()),
		MaxGroupPreparedCapacity: aws.Int64(0),
		MinSize:                  aws.Int64(0),
	}
	if _, err := conn.PutWarmPool(&opts); err != nil {
		return fmt.Errorf("error setting capacity to zero to drain: %s", err)
	}

	// Next, wait for the Warm Pool to drain
	log.Printf("[DEBUG] Waiting for warm pool to have zero instances")
	var p *autoscaling.DescribeWarmPoolOutput
	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p, err := getGroupWarmPool(d.Id(), conn)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if len(p.Instances) == 0 {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("Warm pool still has %d instances", len(p.Instances)))
	})

	if tfresource.TimedOut(err) {
		p, err = getGroupWarmPool(d.Id(), conn)
		if err != nil {
			return fmt.Errorf("error getting Warm Pool info when draining Auto Scaling Group %s: %s", d.Id(), err)
		}
		if p != nil && len(p.Instances) > 0 {
			return fmt.Errorf("Warm pool still has %d instances", len(p.Instances))
		}
	}
	if err != nil {
		return fmt.Errorf("error draining Warm pool: %s", err)
	}
	return nil
}

// TODO: make this a finder
// TODO: this should return a NotFoundError if not found
func getGroup(asgName string, conn *autoscaling.AutoScaling) (*autoscaling.Group, error) {
	describeOpts := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	}

	log.Printf("[DEBUG] Auto Scaling Group describe configuration: %#v", describeOpts)
	describeGroups, err := conn.DescribeAutoScalingGroups(&describeOpts)
	if err != nil {
		autoscalingerr, ok := err.(awserr.Error)
		if ok && autoscalingerr.Code() == "InvalidGroup.NotFound" {
			return nil, nil
		}

		return nil, fmt.Errorf("Error retrieving Auto Scaling Groups: %s", err)
	}

	// Search for the Auto Scaling Group
	for idx, asc := range describeGroups.AutoScalingGroups {
		if asc == nil {
			continue
		}

		if aws.StringValue(asc.AutoScalingGroupName) == asgName {
			return describeGroups.AutoScalingGroups[idx], nil
		}
	}

	return nil, nil
}

func resourceGroupDrain(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn

	if d.Get("force_delete").(bool) {
		log.Printf("[DEBUG] Skipping Auto Scaling Group drain, force_delete was set.")
		return nil
	}

	// First, set the capacity to zero so the group will drain
	log.Printf("[DEBUG] Reducing Auto Scaling Group capacity to zero")
	opts := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(d.Id()),
		DesiredCapacity:      aws.Int64(0),
		MinSize:              aws.Int64(0),
		MaxSize:              aws.Int64(0),
	}
	if _, err := conn.UpdateAutoScalingGroup(&opts); err != nil {
		return fmt.Errorf("Error setting capacity to zero to drain: %s", err)
	}

	// Next, ensure that instances are not prevented from scaling in.
	//
	// The ASG's own scale-in protection setting doesn't make a difference here,
	// as it only affects new instances, which won't be launched now that the
	// desired capacity is set to 0. There is also the possibility that this ASG
	// no longer applies scale-in protection to new instances, but there's still
	// old ones that have it.
	log.Printf("[DEBUG] Disabling scale-in protection for all instances in the group")
	if err := disableASGScaleInProtections(d, conn); err != nil {
		return fmt.Errorf("Error disabling scale-in protection for all instances: %s", err)
	}

	// Next, wait for the Auto Scaling Group to drain
	log.Printf("[DEBUG] Waiting for group to have zero instances")
	var g *autoscaling.Group
	err := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		g, err := getGroup(d.Id(), conn)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if g == nil {
			return nil
		}

		if len(g.Instances) == 0 {
			return nil
		}

		return resource.RetryableError(
			fmt.Errorf("Group still has %d instances", len(g.Instances)))
	})
	if tfresource.TimedOut(err) {
		g, err = getGroup(d.Id(), conn)
		if err != nil {
			return fmt.Errorf("Error getting Auto Scaling Group info when draining: %s", err)
		}
		if g != nil && len(g.Instances) > 0 {
			return fmt.Errorf("Group still has %d instances", len(g.Instances))
		}
	}
	if err != nil {
		return fmt.Errorf("Error draining Auto Scaling Group: %s", err)
	}
	return nil
}

func enableASGSuspendedProcesses(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {
	props := &autoscaling.ScalingProcessQuery{
		AutoScalingGroupName: aws.String(d.Id()),
		ScalingProcesses:     flex.ExpandStringSet(d.Get("suspended_processes").(*schema.Set)),
	}

	_, err := conn.SuspendProcesses(props)
	return err
}

func enableASGMetricsCollection(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {
	props := &autoscaling.EnableMetricsCollectionInput{
		AutoScalingGroupName: aws.String(d.Id()),
		Granularity:          aws.String(d.Get("metrics_granularity").(string)),
		Metrics:              flex.ExpandStringSet(d.Get("enabled_metrics").(*schema.Set)),
	}

	log.Printf("[INFO] Enabling metrics collection for the Auto Scaling Group: %s", d.Id())
	_, metricsErr := conn.EnableMetricsCollection(props)
	return metricsErr

}

func updateASGSuspendedProcesses(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {
	o, n := d.GetChange("suspended_processes")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	resumeProcesses := os.Difference(ns)
	if resumeProcesses.Len() != 0 {
		props := &autoscaling.ScalingProcessQuery{
			AutoScalingGroupName: aws.String(d.Id()),
			ScalingProcesses:     flex.ExpandStringSet(resumeProcesses),
		}

		_, err := conn.ResumeProcesses(props)
		if err != nil {
			return fmt.Errorf("Error Resuming Processes for Auto Scaling Group %q: %s", d.Id(), err)
		}
	}

	suspendedProcesses := ns.Difference(os)
	if suspendedProcesses.Len() != 0 {
		props := &autoscaling.ScalingProcessQuery{
			AutoScalingGroupName: aws.String(d.Id()),
			ScalingProcesses:     flex.ExpandStringSet(suspendedProcesses),
		}

		_, err := conn.SuspendProcesses(props)
		if err != nil {
			return fmt.Errorf("Error Suspending Processes for Auto Scaling Group %q: %s", d.Id(), err)
		}
	}

	return nil

}

func updateASGMetricsCollection(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {

	o, n := d.GetChange("enabled_metrics")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	disableMetrics := os.Difference(ns)
	if disableMetrics.Len() != 0 {
		props := &autoscaling.DisableMetricsCollectionInput{
			AutoScalingGroupName: aws.String(d.Id()),
			Metrics:              flex.ExpandStringSet(disableMetrics),
		}

		_, err := conn.DisableMetricsCollection(props)
		if err != nil {
			return fmt.Errorf("Failure to Disable metrics collection types for Auto Scaling Group %s: %s", d.Id(), err)
		}
	}

	enabledMetrics := ns.Difference(os)
	if enabledMetrics.Len() != 0 {
		props := &autoscaling.EnableMetricsCollectionInput{
			AutoScalingGroupName: aws.String(d.Id()),
			Metrics:              flex.ExpandStringSet(enabledMetrics),
			Granularity:          aws.String(d.Get("metrics_granularity").(string)),
		}

		_, err := conn.EnableMetricsCollection(props)
		if err != nil {
			return fmt.Errorf("Failure to Enable metrics collection types for Auto Scaling Group %s: %s", d.Id(), err)
		}
	}

	return nil
}

// getELBInstanceStates returns a mapping of the instance states of all the ELBs attached to the
// provided ASG.
//
// Note that this is the instance state function for ELB Classic.
//
// Nested like: lbName -> instanceId -> instanceState
func getELBInstanceStates(g *autoscaling.Group, meta interface{}) (map[string]map[string]string, error) {
	lbInstanceStates := make(map[string]map[string]string)
	conn := meta.(*conns.AWSClient).ELBConn

	for _, lbName := range g.LoadBalancerNames {
		lbInstanceStates[aws.StringValue(lbName)] = make(map[string]string)
		opts := &elb.DescribeInstanceHealthInput{LoadBalancerName: lbName}
		r, err := conn.DescribeInstanceHealth(opts)
		if err != nil {
			return nil, err
		}
		for _, is := range r.InstanceStates {
			if is == nil || is.InstanceId == nil || is.State == nil {
				continue
			}
			lbInstanceStates[aws.StringValue(lbName)][aws.StringValue(is.InstanceId)] = aws.StringValue(is.State)
		}
	}

	return lbInstanceStates, nil
}

// getTargetGroupInstanceStates returns a mapping of the instance states of
// all the ALB target groups attached to the provided ASG.
//
// Note that this is the instance state function for Application Load
// Balancing (aka ELBv2).
//
// Nested like: targetGroupARN -> instanceId -> instanceState
func getTargetGroupInstanceStates(g *autoscaling.Group, meta interface{}) (map[string]map[string]string, error) {
	targetInstanceStates := make(map[string]map[string]string)
	conn := meta.(*conns.AWSClient).ELBV2Conn

	for _, targetGroupARN := range g.TargetGroupARNs {
		targetInstanceStates[aws.StringValue(targetGroupARN)] = make(map[string]string)
		opts := &elbv2.DescribeTargetHealthInput{TargetGroupArn: targetGroupARN}
		r, err := conn.DescribeTargetHealth(opts)
		if err != nil {
			return nil, err
		}
		for _, desc := range r.TargetHealthDescriptions {
			if desc == nil || desc.Target == nil || desc.Target.Id == nil || desc.TargetHealth == nil || desc.TargetHealth.State == nil {
				continue
			}
			targetInstanceStates[aws.StringValue(targetGroupARN)][aws.StringValue(desc.Target.Id)] = aws.StringValue(desc.TargetHealth.State)
		}
	}

	return targetInstanceStates, nil
}

func expandVpcZoneIdentifiers(list []interface{}) *string {
	strs := make([]string, len(list))
	for i, s := range list {
		strs[i] = s.(string)
	}
	return aws.String(strings.Join(strs, ","))
}

func expandInstancesDistribution(l []interface{}) *autoscaling.InstancesDistribution {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	instancesDistribution := &autoscaling.InstancesDistribution{}

	if v, ok := m["on_demand_allocation_strategy"]; ok && v.(string) != "" {
		instancesDistribution.OnDemandAllocationStrategy = aws.String(v.(string))
	}

	if v, ok := m["on_demand_base_capacity"]; ok {
		instancesDistribution.OnDemandBaseCapacity = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["on_demand_percentage_above_base_capacity"]; ok {
		instancesDistribution.OnDemandPercentageAboveBaseCapacity = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["spot_allocation_strategy"]; ok && v.(string) != "" {
		instancesDistribution.SpotAllocationStrategy = aws.String(v.(string))
	}

	if v, ok := m["spot_instance_pools"]; ok && v.(int) != 0 {
		instancesDistribution.SpotInstancePools = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["spot_max_price"]; ok {
		instancesDistribution.SpotMaxPrice = aws.String(v.(string))
	}

	return instancesDistribution
}

func expandMixedInstancesLaunchTemplate(l []interface{}) *autoscaling.LaunchTemplate {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	launchTemplate := &autoscaling.LaunchTemplate{
		LaunchTemplateSpecification: expandMixedInstancesLaunchTemplateSpecification(m["launch_template_specification"].([]interface{})),
	}

	if v, ok := m["override"]; ok {
		launchTemplate.Overrides = expandLaunchTemplateOverrides(v.([]interface{}))
	}

	return launchTemplate
}

func expandLaunchTemplateOverrides(l []interface{}) []*autoscaling.LaunchTemplateOverrides {
	if len(l) == 0 {
		return nil
	}

	launchTemplateOverrides := make([]*autoscaling.LaunchTemplateOverrides, len(l))
	for i, m := range l {
		if m == nil {
			launchTemplateOverrides[i] = &autoscaling.LaunchTemplateOverrides{}
			continue
		}

		launchTemplateOverrides[i] = expandLaunchTemplateOverride(m.(map[string]interface{}))
	}
	return launchTemplateOverrides
}

func expandLaunchTemplateOverride(m map[string]interface{}) *autoscaling.LaunchTemplateOverrides {
	launchTemplateOverrides := &autoscaling.LaunchTemplateOverrides{}

	if v, ok := m["instance_type"]; ok && v.(string) != "" {
		launchTemplateOverrides.InstanceType = aws.String(v.(string))
	}

	if v, ok := m["launch_template_specification"]; ok && v.([]interface{}) != nil {
		launchTemplateOverrides.LaunchTemplateSpecification = expandMixedInstancesLaunchTemplateSpecification(m["launch_template_specification"].([]interface{}))
	}

	if v, ok := m["weighted_capacity"]; ok && v.(string) != "" {
		launchTemplateOverrides.WeightedCapacity = aws.String(v.(string))
	}

	return launchTemplateOverrides
}

func expandMixedInstancesLaunchTemplateSpecification(l []interface{}) *autoscaling.LaunchTemplateSpecification {
	launchTemplateSpecification := &autoscaling.LaunchTemplateSpecification{}

	if len(l) == 0 || l[0] == nil {
		return launchTemplateSpecification
	}

	m := l[0].(map[string]interface{})

	if v, ok := m["launch_template_id"]; ok && v.(string) != "" {
		launchTemplateSpecification.LaunchTemplateId = aws.String(v.(string))
	}

	// API returns both ID and name, which Terraform saves to state. Next update returns:
	// ValidationError: Valid requests must contain either launchTemplateId or LaunchTemplateName
	// Prefer the ID if we have both.
	if v, ok := m["launch_template_name"]; ok && v.(string) != "" && launchTemplateSpecification.LaunchTemplateId == nil {
		launchTemplateSpecification.LaunchTemplateName = aws.String(v.(string))
	}

	if v, ok := m["version"]; ok && v.(string) != "" {
		launchTemplateSpecification.Version = aws.String(v.(string))
	}

	return launchTemplateSpecification
}

func expandMixedInstancesPolicy(l []interface{}) *autoscaling.MixedInstancesPolicy {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	mixedInstancesPolicy := &autoscaling.MixedInstancesPolicy{
		LaunchTemplate: expandMixedInstancesLaunchTemplate(m["launch_template"].([]interface{})),
	}

	if v, ok := m["instances_distribution"]; ok {
		mixedInstancesPolicy.InstancesDistribution = expandInstancesDistribution(v.([]interface{}))
	}

	return mixedInstancesPolicy
}

func flattenInstancesDistribution(instancesDistribution *autoscaling.InstancesDistribution) []interface{} {
	if instancesDistribution == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"on_demand_allocation_strategy":            aws.StringValue(instancesDistribution.OnDemandAllocationStrategy),
		"on_demand_base_capacity":                  aws.Int64Value(instancesDistribution.OnDemandBaseCapacity),
		"on_demand_percentage_above_base_capacity": aws.Int64Value(instancesDistribution.OnDemandPercentageAboveBaseCapacity),
		"spot_allocation_strategy":                 aws.StringValue(instancesDistribution.SpotAllocationStrategy),
		"spot_instance_pools":                      aws.Int64Value(instancesDistribution.SpotInstancePools),
		"spot_max_price":                           aws.StringValue(instancesDistribution.SpotMaxPrice),
	}

	return []interface{}{m}
}

func flattenLaunchTemplate(launchTemplate *autoscaling.LaunchTemplate) []interface{} {
	if launchTemplate == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"launch_template_specification": flattenLaunchTemplateSpecification(launchTemplate.LaunchTemplateSpecification),
		"override":                      flattenLaunchTemplateOverrides(launchTemplate.Overrides),
	}

	return []interface{}{m}
}

func flattenLaunchTemplateOverrides(launchTemplateOverrides []*autoscaling.LaunchTemplateOverrides) []interface{} {
	l := make([]interface{}, len(launchTemplateOverrides))

	for i, launchTemplateOverride := range launchTemplateOverrides {
		if launchTemplateOverride == nil {
			l[i] = map[string]interface{}{}
			continue
		}
		m := map[string]interface{}{
			"instance_type":                 aws.StringValue(launchTemplateOverride.InstanceType),
			"launch_template_specification": flattenLaunchTemplateSpecification(launchTemplateOverride.LaunchTemplateSpecification),
			"weighted_capacity":             aws.StringValue(launchTemplateOverride.WeightedCapacity),
		}
		l[i] = m
	}

	return l
}

func flattenLaunchTemplateSpecification(launchTemplateSpecification *autoscaling.LaunchTemplateSpecification) []interface{} {
	if launchTemplateSpecification == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"launch_template_id":   aws.StringValue(launchTemplateSpecification.LaunchTemplateId),
		"launch_template_name": aws.StringValue(launchTemplateSpecification.LaunchTemplateName),
		"version":              aws.StringValue(launchTemplateSpecification.Version),
	}

	return []interface{}{m}
}

func flattenMixedInstancesPolicy(mixedInstancesPolicy *autoscaling.MixedInstancesPolicy) []interface{} {
	if mixedInstancesPolicy == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"instances_distribution": flattenInstancesDistribution(mixedInstancesPolicy.InstancesDistribution),
		"launch_template":        flattenLaunchTemplate(mixedInstancesPolicy.LaunchTemplate),
	}

	return []interface{}{m}
}

func FlattenWarmPoolConfiguration(warmPoolConfiguration *autoscaling.WarmPoolConfiguration) []interface{} {
	if warmPoolConfiguration == nil {
		return []interface{}{}
	}

	maxGroupPreparedCapacity := int64(-1)
	if warmPoolConfiguration.MaxGroupPreparedCapacity != nil {
		maxGroupPreparedCapacity = aws.Int64Value(warmPoolConfiguration.MaxGroupPreparedCapacity)
	}

	m := map[string]interface{}{
		"pool_state":                  aws.StringValue(warmPoolConfiguration.PoolState),
		"min_size":                    aws.Int64Value(warmPoolConfiguration.MinSize),
		"max_group_prepared_capacity": maxGroupPreparedCapacity,
	}

	if warmPoolConfiguration.InstanceReusePolicy != nil {
		m["instance_reuse_policy"] = flattenWarmPoolInstanceReusePolicy(warmPoolConfiguration.InstanceReusePolicy)
	}

	return []interface{}{m}
}

func flattenWarmPoolInstanceReusePolicy(instanceReusePolicy *autoscaling.InstanceReusePolicy) []interface{} {
	if instanceReusePolicy == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"reuse_on_scale_in": aws.BoolValue(instanceReusePolicy.ReuseOnScaleIn),
	}

	return []interface{}{m}
}

func waitUntilGroupLoadBalancersAdded(conn *autoscaling.AutoScaling, asgName string) error {
	input := &autoscaling.DescribeLoadBalancersInput{
		AutoScalingGroupName: aws.String(asgName),
	}
	var lbAdding bool

	for {
		output, err := conn.DescribeLoadBalancers(input)

		if err != nil {
			return err
		}

		for _, tg := range output.LoadBalancers {
			if aws.StringValue(tg.State) == "Adding" {
				lbAdding = true
				break
			}
		}

		if lbAdding {
			lbAdding = false
			input.NextToken = nil
			continue
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil
}

func waitUntilGroupLoadBalancersRemoved(conn *autoscaling.AutoScaling, asgName string) error {
	input := &autoscaling.DescribeLoadBalancersInput{
		AutoScalingGroupName: aws.String(asgName),
	}
	var lbRemoving bool

	for {
		output, err := conn.DescribeLoadBalancers(input)

		if err != nil {
			return err
		}

		for _, tg := range output.LoadBalancers {
			if aws.StringValue(tg.State) == "Removing" {
				lbRemoving = true
				break
			}
		}

		if lbRemoving {
			lbRemoving = false
			input.NextToken = nil
			continue
		}

		if aws.StringValue(output.NextToken) == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return nil
}

func CreatePutWarmPoolInput(asgName string, l []interface{}) *autoscaling.PutWarmPoolInput {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	input := autoscaling.PutWarmPoolInput{
		AutoScalingGroupName: aws.String(asgName),
	}

	if v, ok := m["pool_state"]; ok && v.(string) != "" {
		input.PoolState = aws.String(v.(string))
	}

	if v, ok := m["min_size"]; ok && v.(int) > -1 {
		input.MinSize = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["max_group_prepared_capacity"]; ok && v.(int) > -2 {
		input.MaxGroupPreparedCapacity = aws.Int64(int64(v.(int)))
	}

	if v, ok := m["instance_reuse_policy"]; ok && len(v.([]interface{})) > 0 {
		input.InstanceReusePolicy = expandWarmPoolInstanceReusePolicy(v.([]interface{}))
	}

	return &input
}

func CreateGroupInstanceRefreshInput(asgName string, l []interface{}) *autoscaling.StartInstanceRefreshInput {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	return &autoscaling.StartInstanceRefreshInput{
		AutoScalingGroupName: aws.String(asgName),
		Strategy:             aws.String(m["strategy"].(string)),
		Preferences:          expandGroupInstanceRefreshPreferences(m["preferences"].([]interface{})),
	}
}

func expandGroupInstanceRefreshPreferences(l []interface{}) *autoscaling.RefreshPreferences {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	refreshPreferences := &autoscaling.RefreshPreferences{}

	if v, ok := m["checkpoint_delay"]; ok {
		if v, null, _ := nullable.Int(v.(string)).Value(); !null {
			refreshPreferences.CheckpointDelay = aws.Int64(v)
		}
	}

	if l, ok := m["checkpoint_percentages"].([]interface{}); ok && len(l) > 0 {
		p := make([]*int64, len(l))
		for i, v := range l {
			p[i] = aws.Int64(int64(v.(int)))
		}
		refreshPreferences.CheckpointPercentages = p
	}

	if v, ok := m["instance_warmup"]; ok {
		if v, null, _ := nullable.Int(v.(string)).Value(); !null {
			refreshPreferences.InstanceWarmup = aws.Int64(v)
		}
	}

	if v, ok := m["min_healthy_percentage"]; ok {
		refreshPreferences.MinHealthyPercentage = aws.Int64(int64(v.(int)))
	}

	return refreshPreferences
}

func expandWarmPoolInstanceReusePolicy(l []interface{}) *autoscaling.InstanceReusePolicy {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	instanceReusePolicy := &autoscaling.InstanceReusePolicy{}

	if v, ok := m["reuse_on_scale_in"]; ok {
		instanceReusePolicy.ReuseOnScaleIn = aws.Bool(v.(bool))
	}

	return instanceReusePolicy
}

func GroupRefreshInstances(conn *autoscaling.AutoScaling, asgName string, refreshConfig []interface{}) error {
	input := CreateGroupInstanceRefreshInput(asgName, refreshConfig)
	err := resource.Retry(instanceRefreshStartedTimeout, func() *resource.RetryError {
		_, err := conn.StartInstanceRefresh(input)
		if tfawserr.ErrCodeEquals(err, autoscaling.ErrCodeInstanceRefreshInProgressFault) {
			cancelErr := cancelInstanceRefresh(conn, asgName)
			if cancelErr != nil {
				return resource.NonRetryableError(cancelErr)
			}
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if tfresource.TimedOut(err) {
		_, err = conn.StartInstanceRefresh(input)
	}
	if err != nil {
		return fmt.Errorf("error starting Instance Refresh: %w", err)
	}

	return nil
}

func cancelInstanceRefresh(conn *autoscaling.AutoScaling, asgName string) error {
	input := autoscaling.CancelInstanceRefreshInput{
		AutoScalingGroupName: aws.String(asgName),
	}
	output, err := conn.CancelInstanceRefresh(&input)
	if tfawserr.ErrCodeEquals(err, autoscaling.ErrCodeActiveInstanceRefreshNotFoundFault) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error cancelling Instance Refresh on Auto Scaling Group (%s): %w", asgName, err)
	}
	if output == nil {
		return fmt.Errorf("error cancelling Instance Refresh on Auto Scaling Group (%s): empty result", asgName)
	}

	_, err = waitInstanceRefreshCancelled(conn, asgName, aws.StringValue(output.InstanceRefreshId))
	if err != nil {
		return fmt.Errorf("error waiting for cancellation of Instance Refresh (%s) on Auto Scaling Group (%s): %w", aws.StringValue(output.InstanceRefreshId), asgName, err)
	}

	return nil
}

func validateGroupInstanceRefreshTriggerFields(i interface{}, path cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok {
		return diag.Errorf("expected type to be string")
	}

	if v == "launch_configuration" || v == "launch_template" || v == "mixed_instances_policy" {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("'%s' always triggers an instance refresh and can be removed", v),
			},
		}
	}

	schema := ResourceGroup().Schema
	for attr, attrSchema := range schema {
		if v == attr {
			if attrSchema.Computed && !attrSchema.Optional {
				return diag.Errorf("'%s' is a read-only parameter and cannot be used to trigger an instance refresh", v)
			}
			return nil
		}
	}

	return diag.Errorf("'%s' is not a recognized parameter name for aws_autoscaling_group", v)
}

func expandLaunchTemplateSpecification(specs []interface{}) *autoscaling.LaunchTemplateSpecification {
	if len(specs) < 1 {
		return nil
	}

	spec := specs[0].(map[string]interface{})

	idValue, idOk := spec["id"]
	nameValue, nameOk := spec["name"]

	result := &autoscaling.LaunchTemplateSpecification{}

	// DescribeAutoScalingGroups returns both name and id but LaunchTemplateSpecification
	// allows only one of them to be set
	if idOk && idValue != "" {
		result.LaunchTemplateId = aws.String(idValue.(string))
	} else if nameOk && nameValue != "" {
		result.LaunchTemplateName = aws.String(nameValue.(string))
	}

	if v, ok := spec["version"]; ok && v != "" {
		result.Version = aws.String(v.(string))
	}

	return result
}

func flattenLaunchTemplateSpecificationMap(lt *autoscaling.LaunchTemplateSpecification) []map[string]interface{} {
	if lt == nil {
		return []map[string]interface{}{}
	}

	attrs := map[string]interface{}{}
	result := make([]map[string]interface{}, 0)

	// id and name are always returned by DescribeAutoscalingGroups
	attrs["id"] = aws.StringValue(lt.LaunchTemplateId)
	attrs["name"] = aws.StringValue(lt.LaunchTemplateName)

	// version is returned only if it was previously set
	if lt.Version != nil {
		attrs["version"] = aws.StringValue(lt.Version)
	} else {
		attrs["version"] = nil
	}

	result = append(result, attrs)

	return result
}

// disableASGScaleInProtections disables scale-in protection for all instances
// in the given Auto-Scaling Group.
func disableASGScaleInProtections(d *schema.ResourceData, conn *autoscaling.AutoScaling) error {
	g, err := getGroup(d.Id(), conn)
	if err != nil {
		return fmt.Errorf("Error getting group %s: %s", d.Id(), err)
	}

	var instanceIds []string
	for _, instance := range g.Instances {
		if aws.BoolValue(instance.ProtectedFromScaleIn) {
			instanceIds = append(instanceIds, aws.StringValue(instance.InstanceId))
		}
	}

	const chunkSize = 50 // API limit

	for i := 0; i < len(instanceIds); i += chunkSize {
		j := i + chunkSize
		if j > len(instanceIds) {
			j = len(instanceIds)
		}

		input := autoscaling.SetInstanceProtectionInput{
			AutoScalingGroupName: aws.String(d.Id()),
			InstanceIds:          aws.StringSlice(instanceIds[i:j]),
			ProtectedFromScaleIn: aws.Bool(false),
		}

		if _, err := conn.SetInstanceProtection(&input); err != nil {
			return fmt.Errorf("Error disabling scale-in protections: %s", err)
		}
	}

	return nil
}
