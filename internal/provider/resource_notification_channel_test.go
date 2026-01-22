package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func TestAccNotificationChannel_slack(t *testing.T) {
	resourceName := "last9_notification_channel.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckNotificationChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfig_slack(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationChannelExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Slack Channel"),
					resource.TestCheckResourceAttr(resourceName, "type", "slack"),
					resource.TestCheckResourceAttr(resourceName, "send_resolved", "true"),
					resource.TestCheckResourceAttr(resourceName, "global", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"destination"}, // Sensitive field
			},
		},
	})
}

func TestAccNotificationChannel_email(t *testing.T) {
	resourceName := "last9_notification_channel.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckNotificationChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfig_email(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationChannelExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Email Channel"),
					resource.TestCheckResourceAttr(resourceName, "type", "email"),
					resource.TestCheckResourceAttr(resourceName, "send_resolved", "false"),
				),
			},
		},
	})
}

func TestAccNotificationChannel_update(t *testing.T) {
	resourceName := "last9_notification_channel.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckNotificationChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfig_slack(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationChannelExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Slack Channel"),
					resource.TestCheckResourceAttr(resourceName, "send_resolved", "true"),
				),
			},
			{
				Config: testAccNotificationChannelConfig_slackUpdated(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationChannelExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Slack Channel Updated"),
					resource.TestCheckResourceAttr(resourceName, "send_resolved", "false"),
				),
			},
		},
	})
}

func TestAccNotificationChannel_pagerduty(t *testing.T) {
	resourceName := "last9_notification_channel.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckNotificationChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfig_pagerduty(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationChannelExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test PagerDuty Channel"),
					resource.TestCheckResourceAttr(resourceName, "type", "pagerduty"),
				),
			},
		},
	})
}

func TestAccNotificationChannel_genericWebhook(t *testing.T) {
	resourceName := "last9_notification_channel.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckNotificationChannelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationChannelConfig_genericWebhook(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationChannelExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "TF Test Webhook Channel"),
					resource.TestCheckResourceAttr(resourceName, "type", "generic_webhook"),
				),
			},
		},
	})
}

// Helper functions

func testAccCheckNotificationChannelExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no notification channel ID is set")
		}

		return nil
	}
}

func testAccCheckNotificationChannelDestroy(s *terraform.State) error {
	provider := testAccProvider()
	providerConfig := provider.Meta()
	if providerConfig == nil {
		// Provider not configured, skip check
		return nil
	}

	apiClient := providerConfig.(*client.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_notification_channel" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			continue
		}

		_, err = apiClient.GetNotificationDestination(id)
		if err == nil {
			return fmt.Errorf("notification channel %d still exists", id)
		}
	}

	return nil
}

// Configuration helpers

func testAccNotificationChannelConfig_slack() string {
	return testAccProviderConfig() + `
resource "last9_notification_channel" "test" {
  name          = "TF Test Slack Channel"
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
  send_resolved = true
}
`
}

func testAccNotificationChannelConfig_slackUpdated() string {
	return testAccProviderConfig() + `
resource "last9_notification_channel" "test" {
  name          = "TF Test Slack Channel Updated"
  type          = "slack"
  destination   = "https://hooks.slack.com/services/T00000000/B00000000/YYYYYYYYYYYYYYYYYYYYYYYY"
  send_resolved = false
}
`
}

func testAccNotificationChannelConfig_email() string {
	return testAccProviderConfig() + `
resource "last9_notification_channel" "test" {
  name          = "TF Test Email Channel"
  type          = "email"
  destination   = "terraform-test@last9.io"
  send_resolved = false
}
`
}

func testAccNotificationChannelConfig_pagerduty() string {
	return testAccProviderConfig() + `
resource "last9_notification_channel" "test" {
  name          = "TF Test PagerDuty Channel"
  type          = "pagerduty"
  destination   = "test-integration-key-12345"
  send_resolved = true
}
`
}

func testAccNotificationChannelConfig_genericWebhook() string {
	return testAccProviderConfig() + `
resource "last9_notification_channel" "test" {
  name          = "TF Test Webhook Channel"
  type          = "generic_webhook"
  destination   = "https://example.com/webhook"
  send_resolved = true
}
`
}
