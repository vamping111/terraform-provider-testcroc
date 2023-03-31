package opsworks

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func ResourceRDSDBInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceRDSDBInstanceRegister,
		Update: resourceRDSDBInstanceUpdate,
		Delete: resourceRDSDBInstanceDeregister,
		Read:   resourceRDSDBInstanceRead,

		Schema: map[string]*schema.Schema{
			"stack_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rds_db_instance_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"db_password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"db_user": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceRDSDBInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient).OpsWorksConn

	req := &opsworks.UpdateRdsDbInstanceInput{
		RdsDbInstanceArn: aws.String(d.Get("rds_db_instance_arn").(string)),
	}

	requestUpdate := false
	if d.HasChange("db_user") {
		req.DbUser = aws.String(d.Get("db_user").(string))
		requestUpdate = true
	}
	if d.HasChange("db_password") {
		req.DbPassword = aws.String(d.Get("db_password").(string))
		requestUpdate = true
	}

	if requestUpdate {
		log.Printf("[DEBUG] Opsworks RDS DB Instance Modification request: %s", req)

		_, err := client.UpdateRdsDbInstance(req)
		if err != nil {
			return fmt.Errorf("Error updating Opsworks RDS DB instance: %s", err)
		}
	}

	return resourceRDSDBInstanceRead(d, meta)
}

func resourceRDSDBInstanceDeregister(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient).OpsWorksConn

	req := &opsworks.DeregisterRdsDbInstanceInput{
		RdsDbInstanceArn: aws.String(d.Get("rds_db_instance_arn").(string)),
	}

	log.Printf("[DEBUG] Unregistering rds db instance '%s' from stack: %s", d.Get("rds_db_instance_arn"), d.Get("stack_id"))

	_, err := client.DeregisterRdsDbInstance(req)

	if tfawserr.ErrCodeEquals(err, opsworks.ErrCodeResourceNotFoundException) {
		log.Printf("[DEBUG] OpsWorks RDS DB instance (%s) not found to delete; removed from state", d.Id())
		return nil
	}

	if err != nil {
		return fmt.Errorf("deregistering Opsworks RDS DB instance: %s", err)
	}

	return nil
}

func resourceRDSDBInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient).OpsWorksConn

	req := &opsworks.DescribeRdsDbInstancesInput{
		StackId: aws.String(d.Get("stack_id").(string)),
	}

	log.Printf("[DEBUG] Reading OpsWorks registered rds db instances for stack: %s", d.Get("stack_id"))

	resp, err := client.DescribeRdsDbInstances(req)

	if tfawserr.ErrCodeEquals(err, opsworks.ErrCodeResourceNotFoundException) {
		log.Printf("[WARN] OpsWorks RDS DB Instance (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("while describing OpsWorks RDS DB Instance (%s): %w", d.Get("stack_id"), err)
	}

	found := false
	id := ""
	for _, instance := range resp.RdsDbInstances {
		id = fmt.Sprintf("%s%s", *instance.RdsDbInstanceArn, *instance.StackId)

		if fmt.Sprintf("%s%s", d.Get("rds_db_instance_arn").(string), d.Get("stack_id").(string)) == id {
			found = true
			d.SetId(id)
			d.Set("stack_id", instance.StackId)
			d.Set("rds_db_instance_arn", instance.RdsDbInstanceArn)
			d.Set("db_user", instance.DbUser)
		}

	}

	if !found {
		d.SetId("")
		log.Printf("[INFO] The RDS instance '%s' could not be found for stack: '%s'", d.Get("rds_db_instance_arn"), d.Get("stack_id"))
	}

	return nil
}

func resourceRDSDBInstanceRegister(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient).OpsWorksConn

	req := &opsworks.RegisterRdsDbInstanceInput{
		StackId:          aws.String(d.Get("stack_id").(string)),
		RdsDbInstanceArn: aws.String(d.Get("rds_db_instance_arn").(string)),
		DbUser:           aws.String(d.Get("db_user").(string)),
		DbPassword:       aws.String(d.Get("db_password").(string)),
	}

	_, err := client.RegisterRdsDbInstance(req)

	if err != nil {
		return fmt.Errorf("Error registering Opsworks RDS DB instance: %s", err)
	}

	return resourceRDSDBInstanceRead(d, meta)
}
