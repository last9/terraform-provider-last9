package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNotificationDestination_byName(t *testing.T) {
	destName := os.Getenv("LAST9_TEST_NOTIFICATION_DEST_NAME")
	if destName == "" {
		t.Skip("Skipping test - LAST9_TEST_NOTIFICATION_DEST_NAME not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNotificationDestinationConfig_byName(destName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "name", destName),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "id"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "type"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "destination"),
				),
			},
		},
	})
}

func TestAccDataSourceNotificationDestination_byID(t *testing.T) {
	destID := os.Getenv("LAST9_TEST_NOTIFICATION_DEST_ID")
	if destID == "" {
		t.Skip("Skipping test - LAST9_TEST_NOTIFICATION_DEST_ID not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNotificationDestinationConfig_byID(destID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.last9_notification_destination.test", "id", destID),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "name"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "type"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "destination"),
				),
			},
		},
	})
}

func TestAccDataSourceNotificationDestination_attributes(t *testing.T) {
	destName := os.Getenv("LAST9_TEST_NOTIFICATION_DEST_NAME")
	if destName == "" {
		t.Skip("Skipping test - LAST9_TEST_NOTIFICATION_DEST_NAME not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNotificationDestinationConfig_byName(destName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "id"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "name"),
					resource.TestCheckResourceAttrSet("data.last9_notification_destination.test", "type"),
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
	return fmt.Sprintf(`
data "last9_notification_destination" "test" {
  name = "%s"
}
`, name)
}

func testAccDataSourceNotificationDestinationConfig_byID(id string) string {
	return fmt.Sprintf(`
data "last9_notification_destination" "test" {
  id = %s
}
`, id)
}
