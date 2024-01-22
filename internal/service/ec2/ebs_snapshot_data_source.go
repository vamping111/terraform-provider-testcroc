package ec2

import (
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceEBSSnapshot() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceEBSSnapshotRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"data_encryption_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"filter": DataSourceFiltersSchema(),
			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"most_recent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"outpost_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_alias": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owners": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"restorable_by_user_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"snapshot_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"snapshot_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_tier": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tftags.TagsSchemaComputed(),
			"volume_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"volume_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceEBSSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn

	restorableUsers, restorableUsersOk := d.GetOk("restorable_by_user_ids")
	filters, filtersOk := d.GetOk("filter")
	snapshotIds, snapshotIdsOk := d.GetOk("snapshot_ids")
	owners, ownersOk := d.GetOk("owners")

	if !restorableUsersOk && !filtersOk && !snapshotIdsOk && !ownersOk {
		return fmt.Errorf("One of snapshot_ids, filters, restorable_by_user_ids, or owners must be assigned")
	}

	params := &ec2.DescribeSnapshotsInput{}
	if restorableUsersOk {
		params.RestorableByUserIds = flex.ExpandStringList(restorableUsers.([]interface{}))
	}
	if filtersOk {
		params.Filters = BuildFiltersDataSource(filters.(*schema.Set))
	}
	if ownersOk {
		params.OwnerIds = flex.ExpandStringList(owners.([]interface{}))
	}
	if snapshotIdsOk {
		params.SnapshotIds = flex.ExpandStringList(snapshotIds.([]interface{}))
	}

	log.Printf("[DEBUG] Reading EBS Snapshot: %s", params)
	resp, err := conn.DescribeSnapshots(params)
	if err != nil {
		return err
	}

	if len(resp.Snapshots) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(resp.Snapshots) > 1 {
		if !d.Get("most_recent").(bool) {
			return fmt.Errorf("Your query returned more than one result. Please try a more " +
				"specific search criteria, or set `most_recent` attribute to true.")
		}
		sort.Slice(resp.Snapshots, func(i, j int) bool {
			return aws.TimeValue(resp.Snapshots[i].StartTime).Unix() > aws.TimeValue(resp.Snapshots[j].StartTime).Unix()
		})
	}

	//Single Snapshot found so set to state
	return snapshotDescriptionAttributes(d, resp.Snapshots[0], meta)
}

func snapshotDescriptionAttributes(d *schema.ResourceData, snapshot *ec2.Snapshot, meta interface{}) error {
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	d.SetId(aws.StringValue(snapshot.SnapshotId))
	d.Set("snapshot_id", snapshot.SnapshotId)
	d.Set("volume_id", snapshot.VolumeId)
	d.Set("data_encryption_key_id", snapshot.DataEncryptionKeyId)
	d.Set("description", snapshot.Description)
	d.Set("encrypted", snapshot.Encrypted)
	d.Set("kms_key_id", snapshot.KmsKeyId)
	d.Set("volume_size", snapshot.VolumeSize)
	d.Set("state", snapshot.State)
	d.Set("owner_id", snapshot.OwnerId)
	d.Set("owner_alias", snapshot.OwnerAlias)
	d.Set("storage_tier", snapshot.StorageTier)
	d.Set("outpost_arn", snapshot.OutpostArn)

	if err := d.Set("tags", KeyValueTags(snapshot.Tags).IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	snapshotArn := arn.ARN{
		Partition: meta.(*conns.AWSClient).Partition,
		Region:    meta.(*conns.AWSClient).Region,
		Resource:  fmt.Sprintf("snapshot/%s", d.Id()),
		Service:   ec2.ServiceName,
	}.String()

	d.Set("arn", snapshotArn)

	return nil
}
