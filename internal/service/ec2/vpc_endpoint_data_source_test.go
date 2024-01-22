package ec2_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccVPCEndpointDataSource_gatewayBasic(t *testing.T) {
	datasourceName := "data.aws_vpc_endpoint.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCEndpointDataSourceConfig_gatewayBasic(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "vpc_endpoint_type", "Gateway"),
					resource.TestCheckResourceAttrSet(datasourceName, "prefix_list_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "cidr_blocks.#"),
					resource.TestCheckResourceAttr(datasourceName, "route_table_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "subnet_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "network_interface_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "security_group_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "private_dns_enabled", "false"),
					resource.TestCheckResourceAttr(datasourceName, "requester_managed", "false"),
					resource.TestCheckResourceAttr(datasourceName, "tags.%", "0"),
					acctest.CheckResourceAttrAccountID(datasourceName, "owner_id"),
					acctest.MatchResourceAttrRegionalARN(datasourceName, "arn", "ec2", regexp.MustCompile(`vpc-endpoint/vpce-.+`)),
				),
			},
		},
	})
}

func TestAccVPCEndpointDataSource_byID(t *testing.T) {
	datasourceName := "data.aws_vpc_endpoint.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCEndpointDataSourceConfig_byID(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "vpc_endpoint_type", "Gateway"),
					resource.TestCheckResourceAttrSet(datasourceName, "prefix_list_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "cidr_blocks.#"),
					resource.TestCheckResourceAttr(datasourceName, "route_table_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "subnet_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "network_interface_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "security_group_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "private_dns_enabled", "false"),
					resource.TestCheckResourceAttr(datasourceName, "requester_managed", "false"),
					resource.TestCheckResourceAttr(datasourceName, "tags.%", "0"),
					acctest.CheckResourceAttrAccountID(datasourceName, "owner_id"),
					acctest.MatchResourceAttrRegionalARN(datasourceName, "arn", "ec2", regexp.MustCompile(`vpc-endpoint/vpce-.+`)),
				),
			},
		},
	})
}

func TestAccVPCEndpointDataSource_byFilter(t *testing.T) {
	datasourceName := "data.aws_vpc_endpoint.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCEndpointDataSourceConfig_byFilter(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "vpc_endpoint_type", "Gateway"),
					resource.TestCheckResourceAttrSet(datasourceName, "prefix_list_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "cidr_blocks.#"),
					resource.TestCheckResourceAttr(datasourceName, "route_table_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "subnet_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "network_interface_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "security_group_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "private_dns_enabled", "false"),
					resource.TestCheckResourceAttr(datasourceName, "requester_managed", "false"),
					resource.TestCheckResourceAttr(datasourceName, "tags.%", "0"),
					acctest.CheckResourceAttrAccountID(datasourceName, "owner_id"),
					acctest.MatchResourceAttrRegionalARN(datasourceName, "arn", "ec2", regexp.MustCompile(`vpc-endpoint/vpce-.+`)),
				),
			},
		},
	})
}

func TestAccVPCEndpointDataSource_byTags(t *testing.T) {
	datasourceName := "data.aws_vpc_endpoint.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCEndpointDataSourceConfig_byTags(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "vpc_endpoint_type", "Gateway"),
					resource.TestCheckResourceAttrSet(datasourceName, "prefix_list_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "cidr_blocks.#"),
					resource.TestCheckResourceAttr(datasourceName, "route_table_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "subnet_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "network_interface_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "security_group_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "private_dns_enabled", "false"),
					resource.TestCheckResourceAttr(datasourceName, "requester_managed", "false"),
					resource.TestCheckResourceAttr(datasourceName, "tags.%", "3"),
					acctest.CheckResourceAttrAccountID(datasourceName, "owner_id"),
					acctest.MatchResourceAttrRegionalARN(datasourceName, "arn", "ec2", regexp.MustCompile(`vpc-endpoint/vpce-.+`)),
				),
			},
		},
	})
}

func TestAccVPCEndpointDataSource_gatewayWithRouteTableAndTags(t *testing.T) {
	datasourceName := "data.aws_vpc_endpoint.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCEndpointDataSourceConfig_gatewayWithRouteTableAndTags(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "vpc_endpoint_type", "Gateway"),
					resource.TestCheckResourceAttrSet(datasourceName, "prefix_list_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "cidr_blocks.#"),
					resource.TestCheckResourceAttr(datasourceName, "route_table_ids.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "subnet_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "network_interface_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "security_group_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "private_dns_enabled", "false"),
					resource.TestCheckResourceAttr(datasourceName, "requester_managed", "false"),
					resource.TestCheckResourceAttr(datasourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(datasourceName, "tags.Name", rName),
					acctest.CheckResourceAttrAccountID(datasourceName, "owner_id"),
					acctest.MatchResourceAttrRegionalARN(datasourceName, "arn", "ec2", regexp.MustCompile(`vpc-endpoint/vpce-.+`)),
				),
			},
		},
	})
}

func TestAccVPCEndpointDataSource_interface(t *testing.T) {
	datasourceName := "data.aws_vpc_endpoint.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCEndpointDataSourceConfig_interface(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "vpc_endpoint_type", "Interface"),
					resource.TestCheckNoResourceAttr(datasourceName, "prefix_list_id"),
					resource.TestCheckResourceAttr(datasourceName, "cidr_blocks.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "route_table_ids.#", "0"),
					resource.TestCheckResourceAttr(datasourceName, "subnet_ids.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "network_interface_ids.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "security_group_ids.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "private_dns_enabled", "false"),
					resource.TestCheckResourceAttr(datasourceName, "requester_managed", "false"),
					resource.TestCheckResourceAttr(datasourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(datasourceName, "tags.Name", rName),
					acctest.CheckResourceAttrAccountID(datasourceName, "owner_id"),
					acctest.MatchResourceAttrRegionalARN(datasourceName, "arn", "ec2", regexp.MustCompile(`vpc-endpoint/vpce-.+`)),
				),
			},
		},
	})
}

func testAccVPCEndpointDataSourceConfig_gatewayBasic(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

data "aws_region" "current" {}

resource "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.s3"
}

data "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = aws_vpc_endpoint.test.service_name
  state        = "available"
}
`, rName)
}

func testAccVPCEndpointDataSourceConfig_byID(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

data "aws_region" "current" {}

resource "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.s3"
}

data "aws_vpc_endpoint" "test" {
  id = aws_vpc_endpoint.test.id
}
`, rName)
}

func testAccVPCEndpointDataSourceConfig_byFilter(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

data "aws_region" "current" {}

resource "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.s3"
}

data "aws_vpc_endpoint" "test" {
  filter {
    name   = "vpc-endpoint-id"
    values = [aws_vpc_endpoint.test.id]
  }
}
`, rName)
}

func testAccVPCEndpointDataSourceConfig_byTags(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

data "aws_region" "current" {}

resource "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.s3"

  tags = {
    Key1 = "Value1"
    Key2 = "Value2"
    Key3 = "Value3"
  }
}

data "aws_vpc_endpoint" "test" {
  vpc_id = aws_vpc_endpoint.test.vpc_id

  tags = {
    Key1 = "Value1"
    Key2 = "Value2"
    Key3 = "Value3"
  }
}
`, rName)
}

func testAccVPCEndpointDataSourceConfig_gatewayWithRouteTableAndTags(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_route_table" "test" {
  vpc_id = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

data "aws_region" "current" {}

resource "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = "com.amazonaws.${data.aws_region.current.name}.s3"

  route_table_ids = [
    aws_route_table.test.id,
  ]

  tags = {
    Name = %[1]q
  }
}

data "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = aws_vpc_endpoint.test.service_name
  state        = "available"
}
`, rName)
}

func testAccVPCEndpointDataSourceConfig_interface(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_subnet" "test" {
  vpc_id            = aws_vpc.test.id
  cidr_block        = aws_vpc.test.cidr_block
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = %[1]q
  }
}

resource "aws_security_group" "test" {
  vpc_id = aws_vpc.test.id
  name   = %[1]q

  tags = {
    Name = %[1]q
  }
}

data "aws_region" "current" {}

resource "aws_vpc_endpoint" "test" {
  vpc_id              = aws_vpc.test.id
  vpc_endpoint_type   = "Interface"
  service_name        = "com.amazonaws.${data.aws_region.current.name}.ec2"
  private_dns_enabled = false

  subnet_ids = [
    aws_subnet.test.id,
  ]

  security_group_ids = [
    aws_security_group.test.id,
  ]

  tags = {
    Name = %[1]q
  }
}

data "aws_vpc_endpoint" "test" {
  vpc_id       = aws_vpc.test.id
  service_name = aws_vpc_endpoint.test.service_name
  state        = "available"
}
`, rName)
}
