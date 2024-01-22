package ec2_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
)

func TestAccIPAMPoolAllocation_ipv4Basic(t *testing.T) {
	var allocation ec2.IpamPoolAllocation
	resourceName := "aws_vpc_ipam_pool_cidr_allocation.test"
	cidr := "172.2.0.0/28"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t); testAccIPAMPreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVPCIpamPoolAllocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCIpamPoolAllocationIpv4(cidr),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCIpamAllocationExists(resourceName, &allocation),
					resource.TestCheckResourceAttr(resourceName, "cidr", cidr),
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile(`^ipam-pool-alloc-[\da-f]+_ipam-pool(-[\da-f]+)$`)),
					resource.TestMatchResourceAttr(resourceName, "ipam_pool_allocation_id", regexp.MustCompile(`^ipam-pool-alloc-[\da-f]+$`)),
					resource.TestCheckResourceAttrPair(resourceName, "ipam_pool_id", "aws_vpc_ipam_pool.test", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccIPAMPoolAllocation_ipv4BasicNetmask(t *testing.T) {
	var allocation ec2.IpamPoolAllocation
	resourceName := "aws_vpc_ipam_pool_cidr_allocation.test"
	netmask := "28"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t); testAccIPAMPreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVPCIpamPoolAllocationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCIpamPoolAllocationIpv4Netmask(netmask),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVPCIpamAllocationExists(resourceName, &allocation),
					testAccCheckVPCIpamCidrPrefix(&allocation, netmask),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"netmask_length"},
			},
		},
	})
}

func TestAccIPAMPoolAllocation_ipv4DisallowedCidr(t *testing.T) {
	resourceName := "aws_vpc_ipam_pool_cidr_allocation.test"
	disallowedCidr := "172.2.0.0/28"
	netmaskLength := "28"
	expectedCidr := "172.2.0.16/28"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t); testAccIPAMPreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccVPCIpamPoolAllocationIpv4DisallowedCidr(netmaskLength, disallowedCidr),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "cidr", expectedCidr),
					resource.TestCheckResourceAttr(resourceName, "disallowed_cidrs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "disallowed_cidrs.0", disallowedCidr),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrPair(resourceName, "ipam_pool_id", "aws_vpc_ipam_pool.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "netmask_length", netmaskLength),
				),
			},
		},
	})
}

func testAccCheckVPCIpamAllocationExists(n string, allocation *ec2.IpamPoolAllocation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		id := rs.Primary.ID
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn
		cidr_allocation, _, err := tfec2.FindIpamPoolCidrAllocation(conn, id)

		if err != nil {
			return err
		}
		*allocation = *cidr_allocation

		return nil
	}
}

func testAccCheckVPCIpamCidrPrefix(allocation *ec2.IpamPoolAllocation, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if strings.Split(aws.StringValue(allocation.Cidr), "/")[1] != expected {
			return fmt.Errorf("Bad cidr prefix: %s", aws.StringValue(allocation.Cidr))
		}

		return nil
	}
}

func testAccCheckVPCIpamPoolAllocationDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpc_ipam_pool_cidr_allocation" {
			continue
		}

		id := rs.Primary.ID
		_, _, err := tfec2.FindIpamPoolCidrAllocation(conn, id)

		if err != nil {
			if tfawserr.ErrCodeEquals(err, tfec2.IpamPoolAllocationNotFound) || tfawserr.ErrCodeEquals(err, tfec2.InvalidIpamPoolIdNotFound) {
				return nil
			}
			return err
		}

	}

	return nil
}

const testAccVPCIpamPoolCidrPrivateBase = `
data "aws_region" "current" {}

resource "aws_vpc_ipam" "test" {
  description = "test"
  operating_regions {
    region_name = data.aws_region.current.name
  }
}

resource "aws_vpc_ipam_pool" "test" {
  address_family = "ipv4"
  ipam_scope_id  = aws_vpc_ipam.test.private_default_scope_id
  locale         = data.aws_region.current.name
}

resource "aws_vpc_ipam_pool_cidr" "test" {
  ipam_pool_id = aws_vpc_ipam_pool.test.id
  cidr         = "172.2.0.0/24"
}
`

func testAccVPCIpamPoolAllocationIpv4(cidr string) string {
	return acctest.ConfigCompose(
		testAccVPCIpamPoolCidrPrivateBase,
		fmt.Sprintf(`
resource "aws_vpc_ipam_pool_cidr_allocation" "test" {
  ipam_pool_id = aws_vpc_ipam_pool.test.id
  cidr         = %[1]q
  depends_on = [
    aws_vpc_ipam_pool_cidr.test
  ]
}
`, cidr))
}

func testAccVPCIpamPoolAllocationIpv4Netmask(netmask string) string {
	return acctest.ConfigCompose(
		testAccVPCIpamPoolCidrPrivateBase,
		fmt.Sprintf(`
resource "aws_vpc_ipam_pool_cidr_allocation" "test" {
  ipam_pool_id   = aws_vpc_ipam_pool.test.id
  netmask_length = %[1]q
  depends_on = [
    aws_vpc_ipam_pool_cidr.test
  ]
}
`, netmask))
}

func testAccVPCIpamPoolAllocationIpv4DisallowedCidr(netmaskLength, disallowedCidr string) string {
	return acctest.ConfigCompose(
		testAccVPCIpamPoolCidrPrivateBase,
		fmt.Sprintf(`
resource "aws_vpc_ipam_pool_cidr_allocation" "test" {
  ipam_pool_id   = aws_vpc_ipam_pool.test.id
  netmask_length = %[1]q

  disallowed_cidrs = [
    %[2]q
  ]

  depends_on = [
    aws_vpc_ipam_pool_cidr.test
  ]
}
`, netmaskLength, disallowedCidr))
}
