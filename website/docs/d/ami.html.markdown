---
subcategory: "EC2 (Elastic Compute Cloud)"
layout: "aws"
page_title: "AWS: aws_ami"
description: |-
  Get information on an Amazon Machine Image (AMI).
---

# Data Source: aws_ami

Use this data source to get the ID of a registered image for use in other resources.

## Example Usage

```terraform
data "aws_ami" "example" {
  executable_users = ["self"]
  most_recent      = true
  name_regex       = "^example\\d{1}"
  owners           = ["self"]

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}
```

## Argument Reference

* `owners` - (Required) List of image owners to limit search. At least 1 value must be specified.
  Valid values: the CROC Cloud project ID or `self`.
* `most_recent` - (Optional) If more than one result is returned, use the most recent image.
* `executable_users` - (Optional) Limit search to project with *explicit* launch permission on
 the image. Valid items are the CROC Cloud project ID or `self`.

* `filter` - (Optional) One or more name/value pairs to filter.

For more information about filtering, see the [EC2 API documentation][describe-images].

* `name_regex` - (Optional) A regex string to apply to the image list returned
by CROC Cloud. This allows more advanced filtering not supported from the CROC Cloud API. This
filtering is done locally on what CROC Cloud returns, and could have a performance
impact if the result is large. It is recommended to combine this with other
options to narrow down the list CROC Cloud returns.

~> **NOTE:** If more or less than a single match is returned by the search,
Terraform will fail. Ensure that your search is specific enough to return
a single image ID only, or use `most_recent` to choose the most recent one. If
you want to match multiple images, use the [`aws_ami_ids`][tf-ami-ids] data source instead.

## Attributes Reference

`id` is set to the ID of the found image. In addition, the following attributes
are exported:

~> **NOTE:** Some values are not always set and may not be available for
interpolation.

* `arn` - The ARN of the image.
* `architecture` - The OS architecture of the image (ie: `i386` or `x86_64`).
* `block_device_mappings` - Set of objects with block device mappings of the image.
    * `device_name` - The physical name of the device.
    * `ebs` - Map containing EBS information, if the device is EBS based. Unlike most object attributes, these are accessed directly (e.g., `ebs.volume_size` or `ebs["volume_size"]`) rather than accessed through the first element of a list (e.g., `ebs[0].volume_size`).
        * `delete_on_termination` - `true` if the EBS volume will be deleted on termination.
        * `iops` - `0` if the EBS volume is not a provisioned IOPS image, otherwise the supported IOPS count.
        * `snapshot_id` - The ID of the snapshot.
        * `throughput` - The throughput that the volume supports, in MiB/s.
        * `volume_size` - The size of the volume, in GiB.
        * `volume_type` - The volume type.
    * `no_device` - Suppresses the specified device included in the block device mapping of the image.
    * `virtual_name` - The virtual device name (for instance stores).
* `creation_date` - The date and time the image was created.
* `description` - The description of the image that was provided during image
  creation.
* `image_id` - The ID of the image. Should be the same as the resource `id`.
* `image_location` - The location of the image.
* `image_owner_alias` -  The alias of the image owner name.
* `image_type` - The type of image.
* `name` - The name of the image that was provided during image creation.
* `owner_id` - The CROC Cloud project ID.
* `platform` - The value is Windows for `Windows` images; otherwise blank.
* `product_codes` - Any product codes associated with the image.
    * `product_codes.#.product_code_id` - The product code.
    * `product_codes.#.product_code_type` - The type of product code.
* `public` - `true` if the image has public launch permissions.
* `root_device_name` - The device name of the root device.
* `root_device_type` - The type of root device (ie: `ebs` or `instance-store`).
* `root_snapshot_id` - The snapshot id associated with the root device, if any
  (only applies to `ebs` root devices).
* `state` - The current state of the image. If the state is `available`, the image
  is successfully registered and can be used to launch an instance.
* `tags` - Any tags assigned to the image.
* `virtualization_type` - The type of virtualization of the image (ie: `hvm`).

->  **Unsupported attributes**
These exported attributes are currently unsupported by CROC Cloud:

* `boot_mode` - The boot mode of the image. Always `""`.
* `deprecation_time` - The date and time to deprecate the image. Always `""`.
* `ena_support` - Specifies whether enhanced networking with ENA is enabled. Always `false`.
* `ebs_block_device`:
    * `ebs`
        * `encrypted` - Whether the disk is encrypted. Always `false`.
        * `kms_key_id` - The ARN for the KMS encryption key. Always `""`.
        * `outpost_arn` - The ARN of the Outpost. Always `""`.
* `hypervisor` - The hypervisor type of the image. Always `""`.
* `image_location` - Path to an S3 object containing an image manifest. Always `""`.
* `kernel_id` - The id of the kernel image (AKI) that is used as the paravirtual kernel in created instances. Always `""`.
* `platform_details` - The platform details associated with the billing code of the image. Always `""`.
* `ramdisk_id` - (Optional) The id of an initrd image (ARI) that is used when booting the created instances. Always `""`.
* `state_reason` - Describes a state change. Fields are `UNSET` if not available.
    * `state_reason.code` - The reason code for the state change.
    * `state_reason.message` - The message for the state change.
* `sriov_net_support` - When set to `simple`, enables enhanced networking for created instances. Always `""`.
* `usage_operation` - The operation of the Amazon EC2 instance and the billing code that is associated with the image. Always `""`.

[describe-images]: https://docs.cloud.croc.ru/en/api/ec2/images/DescribeImages.html
[tf-ami-ids]: ami_ids.html
