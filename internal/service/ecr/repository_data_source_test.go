package ecr_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/ecr"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccECRRepositoryDataSource_basic(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_ecr_repository.test"
	dataSourceName := "data.aws_ecr_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ecr.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckRepositoryDataSourceConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "arn", dataSourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "registry_id", dataSourceName, "registry_id"),
					resource.TestCheckResourceAttrPair(resourceName, "repository_url", dataSourceName, "repository_url"),
					resource.TestCheckResourceAttrPair(resourceName, "tags", dataSourceName, "tags"),
					resource.TestCheckResourceAttrPair(resourceName, "image_scanning_configuration.#", dataSourceName, "image_scanning_configuration.#"),
					resource.TestCheckResourceAttrPair(resourceName, "image_tag_mutability", dataSourceName, "image_tag_mutability"),
					resource.TestCheckResourceAttrPair(resourceName, "encryption_configuration.#", dataSourceName, "encryption_configuration.#"),
				),
			},
		},
	})
}

func TestAccECRRepositoryDataSource_encryption(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_ecr_repository.test"
	dataSourceName := "data.aws_ecr_repository.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ecr.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckRepositoryDataSourceConfig_encryption(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "arn", dataSourceName, "arn"),
					resource.TestCheckResourceAttrPair(resourceName, "registry_id", dataSourceName, "registry_id"),
					resource.TestCheckResourceAttrPair(resourceName, "repository_url", dataSourceName, "repository_url"),
					resource.TestCheckResourceAttrPair(resourceName, "tags", dataSourceName, "tags"),
					resource.TestCheckResourceAttrPair(resourceName, "image_scanning_configuration.#", dataSourceName, "image_scanning_configuration.#"),
					resource.TestCheckResourceAttrPair(resourceName, "image_tag_mutability", dataSourceName, "image_tag_mutability"),
					resource.TestCheckResourceAttrPair(resourceName, "encryption_configuration.#", dataSourceName, "encryption_configuration.#"),
					resource.TestCheckResourceAttrPair(resourceName, "encryption_configuration.0.encryption_type", dataSourceName, "encryption_configuration.0.encryption_type"),
					resource.TestCheckResourceAttrPair(resourceName, "encryption_configuration.0.kms_key", dataSourceName, "encryption_configuration.0.kms_key"),
				),
			},
		},
	})
}

func TestAccECRRepositoryDataSource_nonExistent(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ecr.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckAWSEcrRepositoryDataSourceConfig_NonExistent,
				ExpectError: regexp.MustCompile(`not found`),
			},
		},
	})
}

const testAccCheckAWSEcrRepositoryDataSourceConfig_NonExistent = `
data "aws_ecr_repository" "test" {
  name = "tf-acc-test-non-existent"
}
`

func testAccCheckRepositoryDataSourceConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_ecr_repository" "test" {
  name = %q

  tags = {
    Environment = "production"
    Usage       = "original"
  }
}

data "aws_ecr_repository" "test" {
  name = aws_ecr_repository.test.name
}
`, rName)
}

func testAccCheckRepositoryDataSourceConfig_encryption(rName string) string {
	return fmt.Sprintf(`
resource "aws_kms_key" "test" {}

resource "aws_ecr_repository" "test" {
  name = %q

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.test.arn
  }
}

data "aws_ecr_repository" "test" {
  name = aws_ecr_repository.test.name
}
`, rName)
}
