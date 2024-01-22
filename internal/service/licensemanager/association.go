package licensemanager

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/licensemanager"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAssociationCreate,
		Read:   resourceAssociationRead,
		Delete: resourceAssociationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"resource_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"license_configuration_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
		},
	}
}

func resourceAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).LicenseManagerConn

	resourceArn := d.Get("resource_arn").(string)
	licenseConfigurationArn := d.Get("license_configuration_arn").(string)

	opts := &licensemanager.UpdateLicenseSpecificationsForResourceInput{
		AddLicenseSpecifications: []*licensemanager.LicenseSpecification{{
			LicenseConfigurationArn: aws.String(licenseConfigurationArn),
		}},
		ResourceArn: aws.String(resourceArn),
	}

	log.Printf("[DEBUG] License Manager association: %s", opts)

	_, err := conn.UpdateLicenseSpecificationsForResource(opts)
	if err != nil {
		return fmt.Errorf("Error creating License Manager association: %s", err)
	}

	d.SetId(fmt.Sprintf("%s,%s", resourceArn, licenseConfigurationArn))

	return resourceAssociationRead(d, meta)
}

func resourceAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).LicenseManagerConn

	resourceArn, licenseConfigurationArn, err := AssociationParseID(d.Id())
	if err != nil {
		return err
	}

	licenseSpecification, err := AssociationFindSpecification(conn, resourceArn, licenseConfigurationArn)
	if err != nil {
		return fmt.Errorf("Error reading License Manager association: %s", err)
	}

	if licenseSpecification == nil {
		log.Printf("[WARN] License Manager association (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("resource_arn", resourceArn)
	d.Set("license_configuration_arn", licenseConfigurationArn)

	return nil
}

func AssociationFindSpecification(conn *licensemanager.LicenseManager, resourceArn, licenseConfigurationArn string) (*licensemanager.LicenseSpecification, error) {
	opts := &licensemanager.ListLicenseSpecificationsForResourceInput{
		ResourceArn: aws.String(resourceArn),
	}

	for {
		resp, err := conn.ListLicenseSpecificationsForResource(opts)

		if err != nil {
			return nil, err
		}

		for _, licenseSpecification := range resp.LicenseSpecifications {
			if aws.StringValue(licenseSpecification.LicenseConfigurationArn) == licenseConfigurationArn {
				return licenseSpecification, nil
			}
		}

		if len(resp.LicenseSpecifications) == 0 || resp.NextToken == nil {
			return nil, nil
		}

		opts.NextToken = resp.NextToken
	}
}

func AssociationParseID(id string) (string, string, error) {
	parts := strings.SplitN(id, ",", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Expected License Manager Association ID in the form resource_arn,license_configuration_arn - received: %s", id)
	}
	return parts[0], parts[1], nil
}

func resourceAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).LicenseManagerConn

	resourceArn, licenseConfigurationArn, err := AssociationParseID(d.Id())
	if err != nil {
		return err
	}

	opts := &licensemanager.UpdateLicenseSpecificationsForResourceInput{
		RemoveLicenseSpecifications: []*licensemanager.LicenseSpecification{{
			LicenseConfigurationArn: aws.String(licenseConfigurationArn),
		}},
		ResourceArn: aws.String(resourceArn),
	}

	log.Printf("[DEBUG] License Manager association: %s", opts)

	_, err = conn.UpdateLicenseSpecificationsForResource(opts)
	if err != nil {
		return fmt.Errorf("Error deleting License Manager association: %s", err)
	}

	return nil
}
