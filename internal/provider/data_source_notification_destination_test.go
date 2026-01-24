package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNotificationDestination_byName(t *testing.T) {
	timestamp := time.Now().UnixNano()
	channelName := fmt.Sprintf("TF Test DS Channel %d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNotificationDestinationConfig_byName(channelName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "name", channelName),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "id"),
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "type", "slack"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "destination"),
				),
			},
		},
	})
}

func TestAccDataSourceNotificationDestination_byID(t *testing.T) {
	timestamp := time.Now().UnixNano()
	channelName := fmt.Sprintf("TF Test DS By ID %d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNotificationDestinationConfig_byID(channelName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.last9_notification_destination.test", "id",
						"last9_notification_channel.test", "id",
					),
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "name", channelName),
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "type", "slack"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "destination"),
				),
			},
		},
	})
}

func TestAccDataSourceNotificationDestination_attributes(t *testing.T) {
	timestamp := time.Now().UnixNano()
	channelName := fmt.Sprintf("TF Test DS Attrs %d", timestamp)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNotificationDestinationConfig_byName(channelName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "id"),
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "name", channelName),
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "type", "slack"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "destination"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "global"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "send_resolved"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "in_use"),
				),
			},
		},
	})
}

// Configuration helpers

func testAccDataSourceNotificationDestinationConfig_byName(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_notification_channel" "test" {
  name          = %q
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/datasource-test"
  send_resolved = true
}

data "last9_notification_destination" "test" {
  name = last9_notification_channel.test.name
}
`, name)
}

func testAccDataSourceNotificationDestinationConfig_byID(name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_notification_channel" "test" {
  name          = %q
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/datasource-byid-test"
  send_resolved = true
}

data "last9_notification_destination" "test" {
  id = last9_notification_channel.test.id
}
`, name)
}
