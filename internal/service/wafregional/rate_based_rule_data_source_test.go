package wafregional_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/service/wafregional"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
)

func TestAccWAFRegionalRateBasedRuleDataSource_basic(t *testing.T) {
	name := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_wafregional_rate_based_rule.wafrule"
	datasourceName := "data.aws_wafregional_rate_based_rule.wafrule"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acctest.PreCheck(t); acctest.PreCheckPartitionHasService(wafregional.EndpointsID, t) },
		ErrorCheck:        acctest.ErrorCheck(t, wafregional.EndpointsID),
		ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRateBasedRuleDataSourceConfig_NonExistent,
				ExpectError: regexp.MustCompile(`WAF Rate Based Rules not found`),
			},
			{
				Config: testAccRateBasedRuleDataSourceConfig_Name(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(datasourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttrPair(datasourceName, "name", resourceName, "name"),
				),
			},
		},
	})
}

func testAccRateBasedRuleDataSourceConfig_Name(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_rate_based_rule" "wafrule" {
  name        = %[1]q
  metric_name = "WafruleTest"
  rate_key    = "IP"
  rate_limit  = 2000
}

data "aws_wafregional_rate_based_rule" "wafrule" {
  name = aws_wafregional_rate_based_rule.wafrule.name
}
`, name)
}

const testAccRateBasedRuleDataSourceConfig_NonExistent = `
data "aws_wafregional_rate_based_rule" "wafrule" {
  name = "tf-acc-test-does-not-exist"
}
`
