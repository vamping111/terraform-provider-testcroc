package ec2_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

// add sweeper to delete known test VPN Gateways

func TestAccVPNSiteGateway_basic(t *testing.T) {
	var v1, v2 ec2.VpnGateway
	resourceName := "aws_vpn_gateway.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	testNotEqual := func(*terraform.State) error {
		if len(v1.VpcAttachments) == 0 {
			return fmt.Errorf("VPN Gateway A is not attached")
		}
		if len(v2.VpcAttachments) == 0 {
			return fmt.Errorf("VPN Gateway B is not attached")
		}

		if aws.StringValue(v1.VpcAttachments[0].VpcId) == aws.StringValue(v2.VpcAttachments[0].VpcId) {
			return fmt.Errorf("Attachment IDs are equal")
		}

		return nil
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVpnGatewayConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v1),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "ec2", regexp.MustCompile(`vpn-gateway/vgw-.+`)),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccVpnGatewayConfigChangeVPC(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v2),
					testNotEqual,
				),
			},
		},
	})
}

func TestAccVPNSiteGateway_withAvailabilityZoneSetToState(t *testing.T) {
	var v ec2.VpnGateway
	resourceName := "aws_vpn_gateway.test"
	azDataSourceName := "data.aws_availability_zones.available"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVpnGatewayConfigWithAZ(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v),
					resource.TestCheckResourceAttrPair(resourceName, "availability_zone", azDataSourceName, "names.0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"availability_zone"},
			},
		},
	})
}

func TestAccVPNSiteGateway_withAmazonSideASN(t *testing.T) {
	var v ec2.VpnGateway
	resourceName := "aws_vpn_gateway.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVpnGatewayConfigWithASN(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v),
					resource.TestCheckResourceAttr(
						resourceName, "amazon_side_asn", "4294967294"),
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

func TestAccVPNSiteGateway_disappears(t *testing.T) {
	var v ec2.VpnGateway
	resourceName := "aws_vpn_gateway.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVpnGatewayConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceVPNGateway(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccVPNSiteGateway_reattach(t *testing.T) {
	var vpc1, vpc2 ec2.Vpc
	var vgw1, vgw2 ec2.VpnGateway
	vpcResourceName1 := "aws_vpc.test1"
	vpcResourceName2 := "aws_vpc.test2"
	resourceName1 := "aws_vpn_gateway.test1"
	resourceName2 := "aws_vpn_gateway.test2"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	testAttachmentFunc := func(vgw *ec2.VpnGateway, vpc *ec2.Vpc) func(*terraform.State) error {
		return func(*terraform.State) error {
			if len(vgw.VpcAttachments) == 0 {
				return fmt.Errorf("VPN Gateway %q has no VPC attachments.", aws.StringValue(vgw.VpnGatewayId))
			}

			if len(vgw.VpcAttachments) > 1 {
				count := 0
				for _, v := range vgw.VpcAttachments {
					if aws.StringValue(v.State) == ec2.AttachmentStatusAttached {
						count += 1
					}
				}
				if count > 1 {
					return fmt.Errorf(
						"VPN Gateway %q has an unexpected number of VPC attachments (more than 1): %#v",
						aws.StringValue(vgw.VpnGatewayId), vgw.VpcAttachments)
				}
			}

			if aws.StringValue(vgw.VpcAttachments[0].State) != ec2.AttachmentStatusAttached {
				return fmt.Errorf("Expected VPN Gateway %q to be attached.", aws.StringValue(vgw.VpnGatewayId))
			}

			if *vgw.VpcAttachments[0].VpcId != *vpc.VpcId {
				return fmt.Errorf("Expected VPN Gateway %q to be attached to VPC %q, but got: %q",
					aws.StringValue(vgw.VpnGatewayId), aws.StringValue(vpc.VpcId), aws.StringValue(vgw.VpcAttachments[0].VpcId))
			}
			return nil
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVpnGatewayConfigReattach(rName),
				Check: resource.ComposeTestCheckFunc(
					acctest.CheckVPCExists(vpcResourceName1, &vpc1),
					acctest.CheckVPCExists(vpcResourceName2, &vpc2),
					testAccCheckVpnGatewayExists(resourceName1, &vgw1),
					testAccCheckVpnGatewayExists(resourceName2, &vgw2),
					testAttachmentFunc(&vgw1, &vpc1),
					testAttachmentFunc(&vgw2, &vpc2),
				),
			},
			{
				ResourceName:      resourceName1,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      resourceName2,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckVpnGatewayConfigReattachChange(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName1, &vgw1),
					testAccCheckVpnGatewayExists(resourceName2, &vgw2),
					testAttachmentFunc(&vgw2, &vpc1),
					testAttachmentFunc(&vgw1, &vpc2),
				),
			},
			{
				Config: testAccCheckVpnGatewayConfigReattach(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName1, &vgw1),
					testAccCheckVpnGatewayExists(resourceName2, &vgw2),
					testAttachmentFunc(&vgw1, &vpc1),
					testAttachmentFunc(&vgw2, &vpc2),
				),
			},
		},
	})
}

func TestAccVPNSiteGateway_tags(t *testing.T) {
	var v ec2.VpnGateway
	resourceName := "aws_vpn_gateway.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVpnGatewayConfigTags1(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckVpnGatewayConfigTags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccCheckVpnGatewayConfigTags1(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVpnGatewayExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func testAccCheckVpnGatewayDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_vpn_gateway" {
			continue
		}

		_, err := tfec2.FindVPNGatewayByID(conn, rs.Primary.ID)

		if tfresource.NotFound(err) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("EC2 VPN Gateway %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccCheckVpnGatewayExists(n string, v *ec2.VpnGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EC2 VPN Gateway ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		output, err := tfec2.FindVPNGatewayByID(conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccVpnGatewayConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test1" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test" {
  vpc_id = aws_vpc.test1.id
}
`, rName)
}

func testAccVpnGatewayConfigChangeVPC(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test1" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc" "test2" {
  cidr_block = "10.2.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test" {
  vpc_id = aws_vpc.test2.id
}
`, rName)
}

func testAccCheckVpnGatewayConfigTags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test" {
  vpc_id = aws_vpc.test.id

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccCheckVpnGatewayConfigTags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test" {
  vpc_id = aws_vpc.test.id

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}

func testAccCheckVpnGatewayConfigReattach(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test1" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc" "test2" {
  cidr_block = "10.2.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test1" {
  vpc_id = aws_vpc.test1.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test2" {
  vpc_id = aws_vpc.test2.id

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccCheckVpnGatewayConfigReattachChange(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test1" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc" "test2" {
  cidr_block = "10.2.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test1" {
  vpc_id = aws_vpc.test2.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test2" {
  vpc_id = aws_vpc.test1.id

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccVpnGatewayConfigWithAZ(rName string) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test" {
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = %[1]q
  }
}
`, rName))
}

func testAccVpnGatewayConfigWithASN(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpn_gateway" "test" {
  vpc_id          = aws_vpc.test.id
  amazon_side_asn = 4294967294

  tags = {
    Name = %[1]q
  }
}
`, rName)
}
