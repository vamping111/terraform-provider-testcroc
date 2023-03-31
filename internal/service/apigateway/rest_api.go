package apigateway

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	awspolicy "github.com/hashicorp/awspolicyequivalence"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceRestAPI() *schema.Resource {
	return &schema.Resource{
		Create: resourceRestAPICreate,
		Read:   resourceRestAPIRead,
		Update: resourceRestAPIUpdate,
		Delete: resourceRestAPIDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"api_key_source": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(apigateway.ApiKeySourceType_Values(), false),
			},

			"policy": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentPolicyDiffs,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},

			"binary_media_types": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"body": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"disable_execute_api_endpoint": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"parameters": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"minimum_compression_size": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      -1,
				ValidateFunc: validation.IntBetween(-1, 10485760),
			},

			"root_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"created_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"execution_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"endpoint_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"types": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									apigateway.EndpointTypeEdge,
									apigateway.EndpointTypeRegional,
									apigateway.EndpointTypePrivate,
								}, false),
							},
						},
						"vpc_endpoint_ids": {
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceRestAPICreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))
	log.Printf("[DEBUG] Creating API Gateway")

	var description *string
	if d.Get("description").(string) != "" {
		description = aws.String(d.Get("description").(string))
	}

	params := &apigateway.CreateRestApiInput{
		Name:        aws.String(d.Get("name").(string)),
		Description: description,
	}

	if v, ok := d.GetOk("endpoint_configuration"); ok {
		params.EndpointConfiguration = expandApiGatewayEndpointConfiguration(v.([]interface{}))
	}

	if v, ok := d.GetOk("api_key_source"); ok {
		params.ApiKeySource = aws.String(v.(string))
	}

	if v, ok := d.GetOk("disable_execute_api_endpoint"); ok {
		params.DisableExecuteApiEndpoint = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("policy"); ok {
		policy, err := structure.NormalizeJsonString(v.(string))

		if err != nil {
			return fmt.Errorf("policy (%s) is invalid JSON: %w", policy, err)
		}

		params.Policy = aws.String(policy)
	}

	if len(tags) > 0 {
		params.Tags = Tags(tags.IgnoreAWS())
	}

	binaryMediaTypes, binaryMediaTypesOk := d.GetOk("binary_media_types")
	if binaryMediaTypesOk {
		params.BinaryMediaTypes = flex.ExpandStringList(binaryMediaTypes.([]interface{}))
	}

	minimumCompressionSize := d.Get("minimum_compression_size").(int)
	if minimumCompressionSize > -1 {
		params.MinimumCompressionSize = aws.Int64(int64(minimumCompressionSize))
	}

	gateway, err := conn.CreateRestApi(params)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway: %s", err)
	}

	d.SetId(aws.StringValue(gateway.Id))

	if body, ok := d.GetOk("body"); ok {
		log.Printf("[DEBUG] Initializing API Gateway from OpenAPI spec %s", d.Id())

		input := &apigateway.PutRestApiInput{
			RestApiId: gateway.Id,
			Mode:      aws.String(apigateway.PutModeOverwrite),
			Body:      []byte(body.(string)),
		}

		if v, ok := d.GetOk("parameters"); ok && len(v.(map[string]interface{})) > 0 {
			input.Parameters = flex.ExpandStringMap(v.(map[string]interface{}))
		}

		output, err := conn.PutRestApi(input)

		if err != nil {
			return fmt.Errorf("error creating API Gateway specification: %s", err)
		}

		// Using PutRestApi with mode overwrite will remove any configuration
		// that was done with CreateRestApi. Reconcile these changes by having
		// any Terraform configured values overwrite imported configuration.

		updateInput := &apigateway.UpdateRestApiInput{
			RestApiId:       aws.String(d.Id()),
			PatchOperations: []*apigateway.PatchOperation{},
		}

		if v, ok := d.GetOk("api_key_source"); ok && v.(string) != aws.StringValue(output.ApiKeySource) {
			updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/apiKeySource"),
				Value: aws.String(v.(string)),
			})
		}

		if v, ok := d.GetOk("binary_media_types"); ok && len(v.([]interface{})) > 0 {
			for _, elem := range aws.StringValueSlice(output.BinaryMediaTypes) {
				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:   aws.String(apigateway.OpRemove),
					Path: aws.String("/binaryMediaTypes/" + escapeJSONPointer(elem)),
				})
			}

			for _, elem := range v.([]interface{}) {
				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:   aws.String(apigateway.OpAdd),
					Path: aws.String("/binaryMediaTypes/" + escapeJSONPointer(elem.(string))),
				})
			}
		}

		if v, ok := d.GetOk("description"); ok && v.(string) != aws.StringValue(output.Description) {
			updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/description"),
				Value: aws.String(v.(string)),
			})
		}

		if v, ok := d.GetOk("disable_execute_api_endpoint"); ok && v.(bool) != aws.BoolValue(output.DisableExecuteApiEndpoint) {
			updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/disableExecuteApiEndpoint"),
				Value: aws.String(strconv.FormatBool(v.(bool))),
			})
		}

		if v, ok := d.GetOk("endpoint_configuration"); ok {
			endpointConfiguration := expandApiGatewayEndpointConfiguration(v.([]interface{}))

			if endpointConfiguration != nil && len(endpointConfiguration.VpcEndpointIds) > 0 {
				if output.EndpointConfiguration != nil {
					for _, elem := range output.EndpointConfiguration.VpcEndpointIds {
						updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
							Op:    aws.String(apigateway.OpRemove),
							Path:  aws.String("/endpointConfiguration/vpcEndpointIds"),
							Value: elem,
						})
					}
				}

				for _, elem := range endpointConfiguration.VpcEndpointIds {
					updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
						Op:    aws.String(apigateway.OpAdd),
						Path:  aws.String("/endpointConfiguration/vpcEndpointIds"),
						Value: elem,
					})
				}
			}
		}

		if v := d.Get("minimum_compression_size").(int); v > -1 && int64(v) != aws.Int64Value(output.MinimumCompressionSize) {
			updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/minimumCompressionSize"),
				Value: aws.String(strconv.Itoa(v)),
			})
		}

		if v, ok := d.GetOk("name"); ok && v.(string) != aws.StringValue(output.Name) {
			updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/name"),
				Value: aws.String(v.(string)),
			})
		}

		if v, ok := d.GetOk("policy"); ok {
			if equivalent, err := awspolicy.PoliciesAreEquivalent(v.(string), aws.StringValue(output.Policy)); err != nil || !equivalent {
				policy, _ := structure.NormalizeJsonString(v.(string)) // validation covers error

				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/policy"),
					Value: aws.String(policy),
				})
			}
		}

		if len(updateInput.PatchOperations) > 0 {
			_, err := conn.UpdateRestApi(updateInput)

			if err != nil {
				return fmt.Errorf("error updating REST API (%s) after OpenAPI import: %w", d.Id(), err)
			}
		}
	}

	return resourceRestAPIRead(d, meta)
}

func resourceRestAPIRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	log.Printf("[DEBUG] Reading API Gateway %s", d.Id())

	api, err := conn.GetRestApi(&apigateway.GetRestApiInput{
		RestApiId: aws.String(d.Id()),
	})
	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, apigateway.ErrCodeNotFoundException) {
		log.Printf("[WARN] API Gateway (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading API Gateway REST API (%s): %s", d.Id(), err)
	}

	getResourcesInput := &apigateway.GetResourcesInput{
		RestApiId: aws.String(d.Id()),
	}
	err = conn.GetResourcesPages(getResourcesInput, func(page *apigateway.GetResourcesOutput, lastPage bool) bool {
		for _, item := range page.Items {
			if aws.StringValue(item.Path) == "/" {
				d.Set("root_resource_id", item.Id)
				return false
			}
		}
		return !lastPage
	})
	if err != nil {
		return fmt.Errorf("error reading API Gateway REST API (%s) resources: %s", d.Id(), err)
	}

	d.Set("name", api.Name)
	d.Set("description", api.Description)
	d.Set("api_key_source", api.ApiKeySource)
	d.Set("disable_execute_api_endpoint", api.DisableExecuteApiEndpoint)

	// The API returns policy as an escaped JSON string
	// {\\\"Version\\\":\\\"2012-10-17\\\",...}
	// The string must be normalized before unquoting as it may contain escaped
	// forward slashes in CIDR blocks, which will break strconv.Unquote

	// I'm not sure why it needs to be wrapped with double quotes first, but it does
	normalized_policy, err := structure.NormalizeJsonString(`"` + aws.StringValue(api.Policy) + `"`)
	if err != nil {
		return fmt.Errorf("error normalizing policy JSON: %w", err)
	}

	policy, err := strconv.Unquote(normalized_policy)
	if err != nil {
		return fmt.Errorf("error unescaping policy: %s", err)
	}

	policyToSet, err := verify.SecondJSONUnlessEquivalent(d.Get("policy").(string), policy)

	if err != nil {
		return fmt.Errorf("while setting policy (%s), encountered: %w", policyToSet, err)
	}

	d.Set("policy", policyToSet)

	d.Set("binary_media_types", api.BinaryMediaTypes)

	execution_arn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Service:   "execute-api",
		Region:    meta.(*conns.AWSClient).Region,
		AccountID: meta.(*conns.AWSClient).AccountID,
		Resource:  d.Id(),
	}.String()
	d.Set("execution_arn", execution_arn)

	if api.MinimumCompressionSize == nil {
		d.Set("minimum_compression_size", -1)
	} else {
		d.Set("minimum_compression_size", api.MinimumCompressionSize)
	}
	if err := d.Set("created_date", api.CreatedDate.Format(time.RFC3339)); err != nil {
		log.Printf("[DEBUG] Error setting created_date: %s", err)
	}

	if err := d.Set("endpoint_configuration", flattenApiGatewayEndpointConfiguration(api.EndpointConfiguration)); err != nil {
		return fmt.Errorf("error setting endpoint_configuration: %s", err)
	}

	tags := KeyValueTags(api.Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	rest_api_arn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Service:   "apigateway",
		Region:    meta.(*conns.AWSClient).Region,
		Resource:  fmt.Sprintf("/restapis/%s", d.Id()),
	}.String()
	d.Set("arn", rest_api_arn)

	return nil
}

func resourceRestAPIUpdateOperations(d *schema.ResourceData) []*apigateway.PatchOperation {
	operations := make([]*apigateway.PatchOperation, 0)

	if d.HasChange("name") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/name"),
			Value: aws.String(d.Get("name").(string)),
		})
	}

	if d.HasChange("description") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	if d.HasChange("api_key_source") {
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/apiKeySource"),
			Value: aws.String(d.Get("api_key_source").(string)),
		})
	}

	if d.HasChange("disable_execute_api_endpoint") {
		value := strconv.FormatBool(d.Get("disable_execute_api_endpoint").(bool))
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/disableExecuteApiEndpoint"),
			Value: aws.String(value),
		})
	}

	if d.HasChange("policy") {
		policy, _ := structure.NormalizeJsonString(d.Get("policy").(string)) // validation covers error

		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/policy"),
			Value: aws.String(policy),
		})
	}

	if d.HasChange("minimum_compression_size") {
		minimumCompressionSize := d.Get("minimum_compression_size").(int)
		var value string
		if minimumCompressionSize > -1 {
			value = strconv.Itoa(minimumCompressionSize)
		}
		operations = append(operations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/minimumCompressionSize"),
			Value: aws.String(value),
		})
	}

	if d.HasChange("binary_media_types") {
		o, n := d.GetChange("binary_media_types")
		prefix := "binaryMediaTypes"

		old := o.([]interface{})
		new := n.([]interface{})

		// Remove every binary media types. Simpler to remove and add new ones,
		// since there are no replacings.
		for _, v := range old {
			operations = append(operations, &apigateway.PatchOperation{
				Op:   aws.String(apigateway.OpRemove),
				Path: aws.String(fmt.Sprintf("/%s/%s", prefix, escapeJSONPointer(v.(string)))),
			})
		}

		// Handle additions
		if len(new) > 0 {
			for _, v := range new {
				operations = append(operations, &apigateway.PatchOperation{
					Op:   aws.String(apigateway.OpAdd),
					Path: aws.String(fmt.Sprintf("/%s/%s", prefix, escapeJSONPointer(v.(string)))),
				})
			}
		}
	}

	if d.HasChange("endpoint_configuration.0.types") {
		// The REST API must have an endpoint type.
		// If attempting to remove the configuration, do nothing.
		if v, ok := d.GetOk("endpoint_configuration"); ok && len(v.([]interface{})) > 0 {
			m := v.([]interface{})[0].(map[string]interface{})

			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpReplace),
				Path:  aws.String("/endpointConfiguration/types/0"),
				Value: aws.String(m["types"].([]interface{})[0].(string)),
			})
		}
	}

	if d.HasChange("endpoint_configuration.0.vpc_endpoint_ids") {
		o, n := d.GetChange("endpoint_configuration.0.vpc_endpoint_ids")
		prefix := "/endpointConfiguration/vpcEndpointIds"

		old := o.(*schema.Set).List()
		new := n.(*schema.Set).List()

		for _, v := range old {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpRemove),
				Path:  aws.String(prefix),
				Value: aws.String(v.(string)),
			})
		}

		for _, v := range new {
			operations = append(operations, &apigateway.PatchOperation{
				Op:    aws.String(apigateway.OpAdd),
				Path:  aws.String(prefix),
				Value: aws.String(v.(string)),
			})
		}
	}

	return operations
}

func resourceRestAPIUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn
	log.Printf("[DEBUG] Updating API Gateway %s", d.Id())

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")
		if err := UpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return fmt.Errorf("error updating tags: %s", err)
		}
	}

	if d.HasChanges("body", "parameters") {
		if body, ok := d.GetOk("body"); ok {
			log.Printf("[DEBUG] Updating API Gateway from OpenAPI spec: %s", d.Id())

			input := &apigateway.PutRestApiInput{
				RestApiId: aws.String(d.Id()),
				Mode:      aws.String(apigateway.PutModeOverwrite),
				Body:      []byte(body.(string)),
			}

			if v, ok := d.GetOk("parameters"); ok && len(v.(map[string]interface{})) > 0 {
				input.Parameters = flex.ExpandStringMap(v.(map[string]interface{}))
			}

			output, err := conn.PutRestApi(input)

			if err != nil {
				return fmt.Errorf("error updating API Gateway specification: %s", err)
			}

			// Using PutRestApi with mode overwrite will remove any configuration
			// that was done previously. Reconcile these changes by having
			// any Terraform configured values overwrite imported configuration.

			updateInput := &apigateway.UpdateRestApiInput{
				RestApiId:       aws.String(d.Id()),
				PatchOperations: []*apigateway.PatchOperation{},
			}

			if v, ok := d.GetOk("api_key_source"); ok && v.(string) != aws.StringValue(output.ApiKeySource) {
				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/apiKeySource"),
					Value: aws.String(v.(string)),
				})
			}

			if v, ok := d.GetOk("binary_media_types"); ok && len(v.([]interface{})) > 0 {
				for _, elem := range aws.StringValueSlice(output.BinaryMediaTypes) {
					updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
						Op:   aws.String(apigateway.OpRemove),
						Path: aws.String("/binaryMediaTypes/" + escapeJSONPointer(elem)),
					})
				}

				for _, elem := range v.([]interface{}) {
					updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
						Op:   aws.String(apigateway.OpAdd),
						Path: aws.String("/binaryMediaTypes/" + escapeJSONPointer(elem.(string))),
					})
				}
			}

			if v, ok := d.GetOk("description"); ok && v.(string) != aws.StringValue(output.Description) {
				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/description"),
					Value: aws.String(v.(string)),
				})
			}

			if v, ok := d.GetOk("disable_execute_api_endpoint"); ok && v.(bool) != aws.BoolValue(output.DisableExecuteApiEndpoint) {
				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/disableExecuteApiEndpoint"),
					Value: aws.String(strconv.FormatBool(v.(bool))),
				})
			}

			if v, ok := d.GetOk("endpoint_configuration"); ok {
				endpointConfiguration := expandApiGatewayEndpointConfiguration(v.([]interface{}))

				if endpointConfiguration != nil && len(endpointConfiguration.VpcEndpointIds) > 0 {
					if output.EndpointConfiguration != nil {
						for _, elem := range output.EndpointConfiguration.VpcEndpointIds {
							updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
								Op:    aws.String(apigateway.OpRemove),
								Path:  aws.String("/endpointConfiguration/vpcEndpointIds"),
								Value: elem,
							})
						}
					}

					for _, elem := range endpointConfiguration.VpcEndpointIds {
						updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
							Op:    aws.String(apigateway.OpAdd),
							Path:  aws.String("/endpointConfiguration/vpcEndpointIds"),
							Value: elem,
						})
					}
				}
			}

			if v := d.Get("minimum_compression_size").(int); v > -1 && int64(v) != aws.Int64Value(output.MinimumCompressionSize) {
				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/minimumCompressionSize"),
					Value: aws.String(strconv.Itoa(v)),
				})
			}

			if v, ok := d.GetOk("name"); ok && v.(string) != aws.StringValue(output.Name) {
				updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
					Op:    aws.String(apigateway.OpReplace),
					Path:  aws.String("/name"),
					Value: aws.String(v.(string)),
				})
			}

			if v, ok := d.GetOk("policy"); ok {
				if equivalent, err := awspolicy.PoliciesAreEquivalent(v.(string), aws.StringValue(output.Policy)); err != nil || !equivalent {
					policy, _ := structure.NormalizeJsonString(v.(string)) // validation covers error

					updateInput.PatchOperations = append(updateInput.PatchOperations, &apigateway.PatchOperation{
						Op:    aws.String(apigateway.OpReplace),
						Path:  aws.String("/policy"),
						Value: aws.String(policy),
					})
				}
			}

			if len(updateInput.PatchOperations) > 0 {
				_, err := conn.UpdateRestApi(updateInput)

				if err != nil {
					return fmt.Errorf("error updating REST API (%s) after OpenAPI import: %w", d.Id(), err)
				}
			}

			return resourceRestAPIRead(d, meta)
		}
	}

	_, err := conn.UpdateRestApi(&apigateway.UpdateRestApiInput{
		RestApiId:       aws.String(d.Id()),
		PatchOperations: resourceRestAPIUpdateOperations(d),
	})

	if err != nil {
		return fmt.Errorf("error updating REST API (%s): %w", d.Id(), err)
	}

	return resourceRestAPIRead(d, meta)
}

func resourceRestAPIDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).APIGatewayConn

	input := &apigateway.DeleteRestApiInput{
		RestApiId: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting API Gateway: %s", input)
	_, err := conn.DeleteRestApi(input)

	if tfawserr.ErrCodeEquals(err, apigateway.ErrCodeNotFoundException) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting API Gateway (%s): %s", d.Id(), err)
	}

	return nil
}

func expandApiGatewayEndpointConfiguration(l []interface{}) *apigateway.EndpointConfiguration {
	if len(l) == 0 {
		return nil
	}

	m := l[0].(map[string]interface{})

	endpointConfiguration := &apigateway.EndpointConfiguration{
		Types: flex.ExpandStringList(m["types"].([]interface{})),
	}

	if endpointIds, ok := m["vpc_endpoint_ids"]; ok {
		endpointConfiguration.VpcEndpointIds = flex.ExpandStringSet(endpointIds.(*schema.Set))
	}

	return endpointConfiguration
}

func flattenApiGatewayEndpointConfiguration(endpointConfiguration *apigateway.EndpointConfiguration) []interface{} {
	if endpointConfiguration == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"types": flex.FlattenStringList(endpointConfiguration.Types),
	}

	if len(endpointConfiguration.VpcEndpointIds) > 0 {
		m["vpc_endpoint_ids"] = aws.StringValueSlice(endpointConfiguration.VpcEndpointIds)
	}

	return []interface{}{m}
}
