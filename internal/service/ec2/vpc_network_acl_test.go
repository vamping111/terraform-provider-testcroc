package ec2_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

func TestAccVPCNetworkACL_basic(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	vpcResourceName := "aws_vpc.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					acctest.MatchResourceAttrRegionalARN(resourceName, "arn", "ec2", regexp.MustCompile(`network-acl/acl-.+`)),
					resource.TestCheckResourceAttr(resourceName, "egress.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "0"),
					acctest.CheckResourceAttrAccountID(resourceName, "owner_id"),
					resource.TestCheckResourceAttr(resourceName, "subnet_ids.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "0"),
					resource.TestCheckResourceAttrPair(resourceName, "vpc_id", vpcResourceName, "id"),
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

func TestAccVPCNetworkACL_disappears(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					acctest.CheckResourceDisappears(acctest.Provider, tfec2.ResourceNetworkACL(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccVPCNetworkACL_tags(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLTags1Config(rName, "key1", "value1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
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
				Config: testAccNetworkACLTags2Config(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccNetworkACLTags1Config(rName, "key2", "value2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func TestAccVPCNetworkACL_Egress_mode(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLEgressModeBlocksConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "egress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLEgressModeNoBlocksConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "egress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLEgressModeZeroedConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "egress.#", "0"),
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

func TestAccVPCNetworkACL_Ingress_mode(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLIngressModeBlocksConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLIngressModeNoBlocksConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "2"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLIngressModeZeroedConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "0"),
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

func TestAccVPCNetworkACL_egressAndIngressRules(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLEgressNIngressConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "1",
						"from_port":  "80",
						"to_port":    "80",
						"action":     "allow",
						"cidr_block": "10.3.0.0/18",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "egress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "2",
						"from_port":  "443",
						"to_port":    "443",
						"action":     "allow",
						"cidr_block": "10.3.0.0/18",
					}),
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

func TestAccVPCNetworkACL_OnlyIngressRules_basic(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLIngressConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "2",
						"from_port":  "443",
						"to_port":    "443",
						"action":     "deny",
						"cidr_block": "10.2.0.0/18",
					}),
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

func TestAccVPCNetworkACL_OnlyIngressRules_update(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLIngressConfig(resourceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":  "6",
						"rule_no":   "1",
						"from_port": "0",
						"to_port":   "22",
						"action":    "deny",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"cidr_block": "10.2.0.0/18",
						"from_port":  "443",
						"rule_no":    "2",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLIngressChangeConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":   "6",
						"rule_no":    "1",
						"from_port":  "0",
						"to_port":    "22",
						"action":     "deny",
						"cidr_block": "10.2.0.0/18",
					}),
				),
			},
		},
	})
}

func TestAccVPCNetworkACL_caseSensitivityNoChanges(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLCaseSensitiveConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
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

func TestAccVPCNetworkACL_onlyEgressRules(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLEgressConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
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

func TestAccVPCNetworkACL_subnetChange(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLSubnetConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "subnet_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test1", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLSubnetChangeConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "subnet_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test2", "id"),
				),
			},
		},
	})

}

func TestAccVPCNetworkACL_subnets(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLSubnet_SubnetIDs(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "subnet_ids.#", "2"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test1", "id"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test2", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLSubnet_SubnetIDsUpdate(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "subnet_ids.#", "3"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test1", "id"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test3", "id"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test4", "id"),
				),
			},
		},
	})
}

func TestAccVPCNetworkACL_subnetsDelete(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLSubnet_SubnetIDs(resourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "subnet_ids.#", "2"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test1", "id"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test2", "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccNetworkACLSubnet_SubnetIDsDeleteOne(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "subnet_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "subnet_ids.*", "aws_subnet.test1", "id"),
				),
			},
		},
	})
}

func TestAccVPCNetworkACL_ipv6Rules(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLIPv6Config(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(
						resourceName, "ingress.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"protocol":        "6",
						"rule_no":         "1",
						"from_port":       "0",
						"to_port":         "22",
						"action":          "allow",
						"ipv6_cidr_block": "::/0",
					}),
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

func TestAccVPCNetworkACL_ipv6ICMPRules(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLIPv6ICMPConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
				),
			},
		},
	})
}

func TestAccVPCNetworkACL_ipv6VPCRules(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLIPv6VPCConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, "ingress.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ingress.*", map[string]string{
						"ipv6_cidr_block": "2600:1f16:d1e:9a00::/56",
					}),
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

func TestAccVPCNetworkACL_espProtocol(t *testing.T) {
	var v ec2.NetworkAcl
	resourceName := "aws_network_acl.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t) },
		ErrorCheck:        acctest.ErrorCheck(t, ec2.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		CheckDestroy:      testAccCheckNetworkACLDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkACLEsp(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkACLExists(resourceName, &v),
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

func testAccCheckNetworkACLDestroy(s *terraform.State) error {
	conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_network_acl" {
			continue
		}

		_, err := tfec2.FindNetworkACLByID(conn, rs.Primary.ID)

		if tfresource.NotFound(err) {
			continue
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("EC2 Network ACL %s still exists", rs.Primary.ID)
	}

	return nil
}

func testAccCheckNetworkACLExists(n string, v *ec2.NetworkAcl) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EC2 Network ACL ID is set: %s", n)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Conn

		output, err := tfec2.FindNetworkACLByID(conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccNetworkACLConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id
}
`, rName)
}

func testAccNetworkACLIPv6ICMPConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  ingress {
    action          = "allow"
    from_port       = 0
    icmp_code       = -1
    icmp_type       = -1
    ipv6_cidr_block = "::/0"
    protocol        = 58
    rule_no         = 1
    to_port         = 0
  }

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLIPv6Config(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  ingress {
    protocol        = 6
    rule_no         = 1
    action          = "allow"
    ipv6_cidr_block = "::/0"
    from_port       = 0
    to_port         = 22
  }

  subnet_ids = [aws_subnet.test.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLIPv6VPCConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.1.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  ingress {
    protocol        = 6
    rule_no         = 1
    action          = "allow"
    ipv6_cidr_block = "2600:1f16:d1e:9a00::/56"
    from_port       = 0
    to_port         = 22
  }

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLIngressConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "deny"
    cidr_block = "10.2.0.0/18"
    from_port  = 0
    to_port    = 22
  }

  ingress {
    protocol   = 6
    rule_no    = 2
    action     = "deny"
    cidr_block = "10.2.0.0/18"
    from_port  = 443
    to_port    = 443
  }

  subnet_ids = [aws_subnet.test.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLCaseSensitiveConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "Allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 0
    to_port    = 22
  }

  subnet_ids = [aws_subnet.test.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLIngressChangeConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "deny"
    cidr_block = "10.2.0.0/18"
    from_port  = 0
    to_port    = 22
  }

  subnet_ids = [aws_subnet.test.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLEgressConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.2.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block              = "10.2.0.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  egress {
    protocol   = 6
    rule_no    = 2
    action     = "allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 443
    to_port    = 443
  }

  egress {
    protocol   = "-1"
    rule_no    = 4
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  egress {
    protocol   = 6
    rule_no    = 1
    action     = "allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 80
    to_port    = 80
  }

  egress {
    protocol   = 6
    rule_no    = 3
    action     = "allow"
    cidr_block = "10.2.0.0/18"
    from_port  = 22
    to_port    = 22
  }

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLEgressNIngressConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.3.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block              = "10.3.0.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  egress {
    protocol   = 6
    rule_no    = 2
    action     = "allow"
    cidr_block = "10.3.0.0/18"
    from_port  = 443
    to_port    = 443
  }

  ingress {
    protocol   = 6
    rule_no    = 1
    action     = "allow"
    cidr_block = "10.3.0.0/18"
    from_port  = 80
    to_port    = 80
  }

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLSubnetConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test1" {
  cidr_block              = "10.1.111.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test2" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test1" {
  vpc_id     = aws_vpc.test.id
  subnet_ids = [aws_subnet.test2.id]

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.test.id
  subnet_ids = [aws_subnet.test1.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLSubnetChangeConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test1" {
  cidr_block              = "10.1.111.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test2" {
  cidr_block              = "10.1.1.0/24"
  vpc_id                  = aws_vpc.test.id
  map_public_ip_on_launch = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.test.id
  subnet_ids = [aws_subnet.test2.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLSubnet_SubnetIDs(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test1" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test2" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.test.id
  subnet_ids = [aws_subnet.test1.id, aws_subnet.test2.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLSubnet_SubnetIDsUpdate(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test1" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test2" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test3" {
  cidr_block = "10.1.222.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test4" {
  cidr_block = "10.1.4.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id
  subnet_ids = [
    aws_subnet.test1.id,
    aws_subnet.test3.id,
    aws_subnet.test4.id,
  ]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLSubnet_SubnetIDsDeleteOne(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test1" {
  cidr_block = "10.1.111.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id     = aws_vpc.test.id
  subnet_ids = [aws_subnet.test1.id]

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLEsp(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  egress {
    protocol   = "esp"
    rule_no    = 5
    action     = "allow"
    cidr_block = "10.3.0.0/18"
    from_port  = 0
    to_port    = 0
  }

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccNetworkACLEgressModeBlocksConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = %[1]q
  }

  vpc_id = aws_vpc.test.id

  egress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = 6
    rule_no    = 1
    to_port    = 0
  }

  egress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = "udp"
    rule_no    = 2
    to_port    = 0
  }
}
`, rName)
}

func testAccNetworkACLEgressModeNoBlocksConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = %[1]q
  }

  vpc_id = aws_vpc.test.id
}
`, rName)
}

func testAccNetworkACLEgressModeZeroedConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  egress = []

  tags = {
    Name = %[1]q
  }

  vpc_id = aws_vpc.test.id
}
`, rName)
}

func testAccNetworkACLIngressModeBlocksConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = %[1]q
  }

  vpc_id = aws_vpc.test.id

  ingress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = 6
    rule_no    = 1
    to_port    = 0
  }

  ingress {
    action     = "allow"
    cidr_block = aws_vpc.test.cidr_block
    from_port  = 0
    protocol   = "udp"
    rule_no    = 2
    to_port    = 0
  }
}
`, rName)
}

func testAccNetworkACLIngressModeNoBlocksConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  tags = {
    Name = %[1]q
  }

  vpc_id = aws_vpc.test.id
}
`, rName)
}

func testAccNetworkACLIngressModeZeroedConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  ingress = []

  tags = {
    Name = %[1]q
  }

  vpc_id = aws_vpc.test.id
}
`, rName)
}

func testAccNetworkACLTags1Config(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccNetworkACLTags2Config(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_network_acl" "test" {
  vpc_id = aws_vpc.test.id

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}
