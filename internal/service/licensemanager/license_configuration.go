package licensemanager

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/licensemanager"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceLicenseConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceLicenseConfigurationCreate,
		Read:   resourceLicenseConfigurationRead,
		Update: resourceLicenseConfigurationUpdate,
		Delete: resourceLicenseConfigurationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"license_count": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"license_count_hard_limit": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"license_counting_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					licensemanager.LicenseCountingTypeVCpu,
					licensemanager.LicenseCountingTypeInstance,
					licensemanager.LicenseCountingTypeCore,
					licensemanager.LicenseCountingTypeSocket,
				}, false),
			},
			"license_rules": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^#([^=]+)=(.+)$"), "Expected format is #RuleType=RuleValue"),
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"owner_account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},

		CustomizeDiff: verify.SetTagsDiff,
	}
}

func resourceLicenseConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).LicenseManagerConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	opts := &licensemanager.CreateLicenseConfigurationInput{
		LicenseCountingType: aws.String(d.Get("license_counting_type").(string)),
		Name:                aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		opts.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("license_count"); ok {
		opts.LicenseCount = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("license_count_hard_limit"); ok {
		opts.LicenseCountHardLimit = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("license_rules"); ok {
		opts.LicenseRules = flex.ExpandStringList(v.([]interface{}))
	}

	if len(tags) > 0 {
		opts.Tags = Tags(tags.IgnoreAWS())
	}

	log.Printf("[DEBUG] License Manager license configuration: %s", opts)

	resp, err := conn.CreateLicenseConfiguration(opts)
	if err != nil {
		return fmt.Errorf("Error creating License Manager license configuration: %s", err)
	}
	d.SetId(aws.StringValue(resp.LicenseConfigurationArn))
	return resourceLicenseConfigurationRead(d, meta)
}

func resourceLicenseConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).LicenseManagerConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	resp, err := conn.GetLicenseConfiguration(&licensemanager.GetLicenseConfigurationInput{
		LicenseConfigurationArn: aws.String(d.Id()),
	})

	if err != nil {
		if tfawserr.ErrCodeEquals(err, licensemanager.ErrCodeInvalidParameterValueException) {
			log.Printf("[WARN] License Manager license configuration (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading License Manager license configuration: %s", err)
	}

	d.Set("arn", resp.LicenseConfigurationArn)
	d.Set("description", resp.Description)
	d.Set("license_count", resp.LicenseCount)
	d.Set("license_count_hard_limit", resp.LicenseCountHardLimit)
	d.Set("license_counting_type", resp.LicenseCountingType)
	if err := d.Set("license_rules", flex.FlattenStringList(resp.LicenseRules)); err != nil {
		return fmt.Errorf("error setting license_rules: %s", err)
	}
	d.Set("name", resp.Name)
	d.Set("owner_account_id", resp.OwnerAccountId)

	tags := KeyValueTags(resp.Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceLicenseConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).LicenseManagerConn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating License Manager License Configuration (%s) tags: %s", d.Id(), err)
		}
	}

	opts := &licensemanager.UpdateLicenseConfigurationInput{
		LicenseConfigurationArn: aws.String(d.Id()),
		Name:                    aws.String(d.Get("name").(string)),
		Description:             aws.String(d.Get("description").(string)),
		LicenseCountHardLimit:   aws.Bool(d.Get("license_count_hard_limit").(bool)),
	}

	if v, ok := d.GetOk("license_count"); ok {
		opts.LicenseCount = aws.Int64(int64(v.(int)))
	}

	log.Printf("[DEBUG] License Manager license configuration: %s", opts)

	_, err := conn.UpdateLicenseConfiguration(opts)
	if err != nil {
		return fmt.Errorf("Error updating License Manager license configuration: %s", err)
	}
	return resourceLicenseConfigurationRead(d, meta)
}

func resourceLicenseConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).LicenseManagerConn

	opts := &licensemanager.DeleteLicenseConfigurationInput{
		LicenseConfigurationArn: aws.String(d.Id()),
	}

	_, err := conn.DeleteLicenseConfiguration(opts)
	if err != nil {
		if tfawserr.ErrCodeEquals(err, licensemanager.ErrCodeInvalidParameterValueException) {
			return nil
		}
		return fmt.Errorf("Error deleting License Manager license configuration: %s", err)
	}

	return nil
}
