package autoscaling

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func ResourceAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAttachmentCreate,
		Read:   resourceAttachmentRead,
		Delete: resourceAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"autoscaling_group_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"elb": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"alb_target_group_arn": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				Deprecated:    "Use lb_target_group_arn instead",
				ConflictsWith: []string{"lb_target_group_arn"},
			},

			"lb_target_group_arn": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"alb_target_group_arn"},
			},
		},
	}
}

func resourceAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn
	asgName := d.Get("autoscaling_group_name").(string)

	if v, ok := d.GetOk("elb"); ok {
		attachOpts := &autoscaling.AttachLoadBalancersInput{
			AutoScalingGroupName: aws.String(asgName),
			LoadBalancerNames:    []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] registering asg %s with ELBs %s", asgName, v.(string))

		if _, err := conn.AttachLoadBalancers(attachOpts); err != nil {
			return fmt.Errorf("Failure attaching AutoScaling Group %s with Elastic Load Balancer: %s: %s", asgName, v.(string), err)
		}
	}

	if v, ok := d.GetOk("alb_target_group_arn"); ok {
		attachOpts := &autoscaling.AttachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: aws.String(asgName),
			TargetGroupARNs:      []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] registering asg %s with ALB Target Group %s", asgName, v.(string))

		if _, err := conn.AttachLoadBalancerTargetGroups(attachOpts); err != nil {
			return fmt.Errorf("failure attaching AutoScaling Group %s with ALB Target Group: %s: %w", asgName, v.(string), err)
		}
	}

	if v, ok := d.GetOk("lb_target_group_arn"); ok {
		attachOpts := &autoscaling.AttachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: aws.String(asgName),
			TargetGroupARNs:      []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] registering asg %s with LB Target Group %s", asgName, v.(string))

		if _, err := conn.AttachLoadBalancerTargetGroups(attachOpts); err != nil {
			return fmt.Errorf("failure attaching AutoScaling Group %s with LB Target Group: %s: %w", asgName, v.(string), err)
		}
	}

	//lintignore:R016 // Allow legacy unstable ID usage in managed resource
	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", asgName)))

	return resourceAttachmentRead(d, meta)
}

func resourceAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn
	asgName := d.Get("autoscaling_group_name").(string)

	// Retrieve the ASG properties to get list of associated ELBs
	asg, err := getGroup(asgName, conn)

	if err != nil {
		return err
	}
	if asg == nil && !d.IsNewResource() {
		log.Printf("[WARN] Autoscaling Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if v, ok := d.GetOk("elb"); ok {
		found := false
		for _, i := range asg.LoadBalancerNames {
			if v.(string) == aws.StringValue(i) {
				d.Set("elb", v.(string))
				found = true
				break
			}
		}

		if !found {
			log.Printf("[WARN] Association for %s was not found in ASG association", v.(string))
			d.SetId("")
		}
	}

	if v, ok := d.GetOk("alb_target_group_arn"); ok {
		found := false
		for _, i := range asg.TargetGroupARNs {
			if v.(string) == aws.StringValue(i) {
				d.Set("alb_target_group_arn", v.(string))
				found = true
				break
			}
		}

		if !found {
			log.Printf("[WARN] Association for %s was not found in ASG association", v.(string))
			d.SetId("")
		}
	}

	if v, ok := d.GetOk("lb_target_group_arn"); ok {
		found := false
		for _, i := range asg.TargetGroupARNs {
			if v.(string) == aws.StringValue(i) {
				d.Set("lb_target_group_arn", v.(string))
				found = true
				break
			}
		}

		if !found {
			log.Printf("[WARN] Association for %s was not found in ASG association", v.(string))
			d.SetId("")
		}
	}

	return nil
}

func resourceAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).AutoScalingConn
	asgName := d.Get("autoscaling_group_name").(string)

	if v, ok := d.GetOk("elb"); ok {
		detachOpts := &autoscaling.DetachLoadBalancersInput{
			AutoScalingGroupName: aws.String(asgName),
			LoadBalancerNames:    []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] Deleting ELB %s association from: %s", v.(string), asgName)
		if _, err := conn.DetachLoadBalancers(detachOpts); err != nil {
			return fmt.Errorf("failure detaching AutoScaling Group %s with Elastic Load Balancer: %s: %w", asgName, v.(string), err)
		}
	}

	if v, ok := d.GetOk("alb_target_group_arn"); ok {
		detachOpts := &autoscaling.DetachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: aws.String(asgName),
			TargetGroupARNs:      []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] Deleting ALB Target Group %s association from: %s", v.(string), asgName)
		if _, err := conn.DetachLoadBalancerTargetGroups(detachOpts); err != nil {
			return fmt.Errorf("failure detaching AutoScaling Group %s with ALB Target Group: %s: %w", asgName, v.(string), err)
		}
	}

	if v, ok := d.GetOk("lb_target_group_arn"); ok {
		detachOpts := &autoscaling.DetachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: aws.String(asgName),
			TargetGroupARNs:      []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] Deleting LB Target Group %s association from: %s", v.(string), asgName)
		if _, err := conn.DetachLoadBalancerTargetGroups(detachOpts); err != nil {
			return fmt.Errorf("failure detaching AutoScaling Group %s with LB Target Group: %s: %w", asgName, v.(string), err)
		}
	}

	return nil
}
