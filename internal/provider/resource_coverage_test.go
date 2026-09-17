package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestParseOTelResourceID(t *testing.T) {
	region, cluster, id, err := parseOTelResourceID("ap-south-1:cluster-1:rule-uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if region != "ap-south-1" || cluster != "cluster-1" || id != "rule-uuid" {
		t.Fatalf("got %s %s %s", region, cluster, id)
	}
	if _, _, _, err := parseOTelResourceID("bad"); err == nil {
		t.Fatal("expected error for bad id")
	}
}

func TestParseRegionID(t *testing.T) {
	region, id, err := parseRegionID("us-east-1:abc")
	if err != nil || region != "us-east-1" || id != "abc" {
		t.Fatalf("got %s %s err=%v", region, id, err)
	}
}

func TestExpandFlattenOTelFilters(t *testing.T) {
	and := "and"
	in := []interface{}{
		map[string]interface{}{
			"key":         `attributes["service"]`,
			"value":       "api",
			"operator":    "equals",
			"conjunction": and,
		},
	}
	expanded := expandOTelFilters(in)
	if len(expanded) != 1 || expanded[0].Key == "" {
		t.Fatalf("expand failed: %+v", expanded)
	}
	flat := flattenOTelFilters(expanded)
	if len(flat) != 1 {
		t.Fatalf("flatten failed: %+v", flat)
	}
}

func TestAccSyntheticCheck_basic(t *testing.T) {
	resourceName := "last9_synthetic_check.test"
	name := fmt.Sprintf("tf-synth-%d", time.Now().Unix())
	location := os.Getenv("LAST9_TEST_SYNTHETIC_LOCATION")
	if location == "" {
		location = "us-east-1"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckSyntheticCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSyntheticCheckConfig(name, location),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "type", "http"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					// API may normalize/mask config JSON
					"config",
				},
			},
		},
	})
}

func TestAccAlertSnooze_basic(t *testing.T) {
	resourceName := "last9_alert_snooze.test"
	entityName := fmt.Sprintf("tf-snooze-%d", time.Now().Unix())
	until := time.Now().Add(2 * time.Hour).Unix()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckAlertSnoozeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertSnoozeConfig(entityName, until),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "entity_id"),
					resource.TestCheckResourceAttr(resourceName, "until", fmt.Sprintf("%d", until)),
				),
			},
		},
	})
}

func TestAccUser_basic(t *testing.T) {
	resourceName := "last9_user.test"
	// Invites only appear in /users when email domain is an allowed org domain.
	domain := os.Getenv("LAST9_TEST_USER_DOMAIN")
	if domain == "" {
		domain = "last9.io"
	}
	email := fmt.Sprintf("tf-test-%d@%s", time.Now().Unix(), domain)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(email, "viewer"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "email", email),
					resource.TestCheckResourceAttr(resourceName, "role", "viewer"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				Config: testAccUserConfig(email, "editor"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "role", "editor"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"active", // computed from deleted_at
				},
			},
		},
	})
}

func TestAccChangeboard_basic(t *testing.T) {
	resourceName := "last9_changeboard.test"
	name := fmt.Sprintf("tf-cb-%d", time.Now().Unix())

	apiClient, err := testAccClientFromEnv()
	if err != nil {
		t.Skip(err.Error())
	}
	users, err := apiClient.ListUsers()
	if err != nil || len(users) == 0 {
		t.Skipf("need at least one user to resolve organization_id: %v", err)
	}
	ownerID := users[0].OrganizationID
	if ownerID == "" {
		t.Skip("organization_id empty on listed user")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckChangeboardDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccChangeboardConfig(name, ownerID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "owner_type", "organization"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccSensitiveDataRule_basic(t *testing.T) {
	resourceName := "last9_sensitive_data_rule.test"
	region := testAccRegion()
	name := fmt.Sprintf("tf-pii-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckSensitiveDataDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSensitiveDataConfig(region, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "action_name", "redact"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccPhysicalIndex_basic(t *testing.T) {
	resourceName := "last9_physical_index.test"
	region := testAccRegion()
	name := fmt.Sprintf("tf_idx_%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckPhysicalIndexDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPhysicalIndexConfig(region, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "telemetry", "logs"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccRehydration_basic(t *testing.T) {
	resourceName := "last9_rehydration.test"
	region := testAccRegion()
	name := fmt.Sprintf("tf-rehydrate-%d", time.Now().Unix())
	// API rejects recent (hot) windows and appears to floor timestamps to the minute.
	to := time.Now().Add(-30 * 24 * time.Hour).Truncate(time.Minute).Unix()
	from := to - 3600

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckRehydrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRehydrationConfig(region, name, from, to),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "physical_index", "logs"),
					resource.TestCheckResourceAttr(resourceName, "telemetry", "logs"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
				),
			},
		},
	})
}

func TestAccStreamingAggregation_basic(t *testing.T) {
	resourceName := "last9_streaming_aggregation.test"
	region := testAccRegion()
	name := fmt.Sprintf("tf-sap-%d", time.Now().Unix())
	outMetric := fmt.Sprintf("tf_sap_out_%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckStreamingAggregationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccStreamingAggregationConfig(region, name, outMetric),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "telemetry", "metrics"),
					resource.TestCheckResourceAttr(resourceName, "aggregation", "sum"),
					resource.TestCheckResourceAttr(resourceName, "clause", "without"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "cluster_id"),
				),
			},
			{
				Config: testAccStreamingAggregationConfigUpdated(region, name, outMetric),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "resolution", "5m"),
					resource.TestCheckResourceAttr(resourceName, "aggregation", "max"),
				),
			},
		},
	})
}

func TestAccColdStorageBucket_basic(t *testing.T) {
	// Creating requires a real, unused S3 bucket the IAM role can access.
	// Do not create against an already-registered aws_bucket (duplicate) and
	// never destroy shared org buckets via import-only tests.
	role := os.Getenv("LAST9_TEST_AWS_ROLE")
	uniq := os.Getenv("LAST9_TEST_AWS_BUCKET_UNIQUE")
	if role == "" || uniq == "" {
		t.Skip("set LAST9_TEST_AWS_ROLE and LAST9_TEST_AWS_BUCKET_UNIQUE (unused S3 bucket) to run create ACC")
	}
	awsRegion := os.Getenv("LAST9_TEST_AWS_REGION")
	if awsRegion == "" {
		awsRegion = "ap-south-1"
	}

	resourceName := "last9_cold_storage_bucket.test"
	region := testAccRegion()
	name := fmt.Sprintf("tf-csb-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckColdStorageBucketDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccColdStorageBucketConfig(region, name, awsRegion, uniq, role),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "auth_type", "role"),
					resource.TestCheckResourceAttr(resourceName, "aws_bucket", uniq),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccColdStorageBackup_basic(t *testing.T) {
	bucketName := os.Getenv("LAST9_TEST_COLD_STORAGE_BUCKET_NAME")
	if bucketName == "" {
		t.Skip("set LAST9_TEST_COLD_STORAGE_BUCKET_NAME to an existing Last9 cold storage bucket name")
	}

	resourceName := "last9_cold_storage_backup.test"
	region := testAccRegion()
	name := fmt.Sprintf("tf-csbackup-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckColdStorageBackupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccColdStorageBackupConfig(region, name, bucketName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "granularity", "index"),
					resource.TestCheckResourceAttr(resourceName, "bucket_name", bucketName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccS3Ingest_basic(t *testing.T) {
	role := os.Getenv("LAST9_TEST_AWS_ROLE")
	bucket := os.Getenv("LAST9_TEST_AWS_BUCKET")
	awsRegion := os.Getenv("LAST9_TEST_AWS_REGION")
	if role == "" || bucket == "" {
		t.Skip("set LAST9_TEST_AWS_ROLE and LAST9_TEST_AWS_BUCKET to run s3 ingest ACC")
	}
	if awsRegion == "" {
		awsRegion = "ap-south-1"
	}

	resourceName := "last9_s3_ingest.test"
	region := testAccRegion()
	name := fmt.Sprintf("tf-s3ingest-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckWithDelete(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckS3IngestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccS3IngestConfig(region, name, awsRegion, bucket, role),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "auth_type", "role"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func TestAccClusterDataSource_default(t *testing.T) {
	region := testAccRegion()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
data "last9_cluster" "default" {
  region = %q
}
`, region),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.last9_cluster.default", "id"),
					resource.TestCheckResourceAttrSet("data.last9_cluster.default", "name"),
				),
			},
		},
	})
}

func TestAccDatasourceDataSource_default(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
data "last9_datasource" "default" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.last9_datasource.default", "id"),
				),
			},
		},
	})
}

func testAccCheckSyntheticCheckDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_synthetic_check" {
			continue
		}
		if _, getErr := c.GetSyntheticCheck(rs.Primary.ID); getErr == nil {
			return fmt.Errorf("synthetic check %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckAlertSnoozeDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_alert_snooze" {
			continue
		}
		resp, getErr := c.GetEntitySnooze(rs.Primary.ID)
		if getErr == nil && resp.AlertSnoozedUntil > int(time.Now().Unix()) {
			return fmt.Errorf("snooze still active for %s until %d", rs.Primary.ID, resp.AlertSnoozedUntil)
		}
	}
	return nil
}

func testAccCheckUserDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_user" {
			continue
		}
		if _, getErr := c.FindUserByID(rs.Primary.ID); getErr == nil {
			return fmt.Errorf("user %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckChangeboardDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_changeboard" {
			continue
		}
		if _, getErr := c.GetChangeBoard(rs.Primary.ID); getErr == nil {
			return fmt.Errorf("changeboard %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckSensitiveDataDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_sensitive_data_rule" {
			continue
		}
		region, id, parseErr := parseRegionID(rs.Primary.ID)
		if parseErr != nil {
			continue
		}
		if _, getErr := c.GetSensitiveData(id, region); getErr == nil {
			return fmt.Errorf("sensitive data rule %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckPhysicalIndexDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_physical_index" {
			continue
		}
		region, clusterID, id, parseErr := parseOTelResourceID(rs.Primary.ID)
		if parseErr != nil {
			continue
		}
		resp, getErr := c.GetPhysicalIndex(id, region)
		if getErr != nil {
			continue // gone
		}
		// Pending indexes cannot be deleted by API; accept as TF-destroyed.
		if strings.Contains(strings.ToLower(resp.Status), "pending") ||
			strings.EqualFold(resp.Status, "creating") {
			continue
		}
		if delErr := c.DeletePhysicalIndex(id, region, clusterID); delErr != nil {
			if strings.Contains(strings.ToLower(delErr.Error()), "not created yet") {
				continue
			}
		} else {
			continue
		}
		return fmt.Errorf("physical index %s still exists", rs.Primary.ID)
	}
	return nil
}

func testAccSyntheticCheckConfig(name, location string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_synthetic_check" "test" {
  name      = %q
  type      = "http"
  schedule  = "every 5m"
  timeout   = 30
  frequency = 60
  locations = [%q]

  config = jsonencode({
    url    = "https://example.com/health"
    method = "GET"
  })
}
`, name, location)
}

func testAccAlertSnoozeConfig(entityName string, until int64) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_entity" "snooze_group" {
  name         = %q
  type         = "service"
  external_ref = %q
}

resource "last9_alert_snooze" "test" {
  entity_id = last9_entity.snooze_group.id
  until     = %d
}
`, entityName, entityName, until)
}

func testAccUserConfig(email, role string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_user" "test" {
  email  = %q
  role   = %q
  active = true
}
`, email, role)
}

func testAccChangeboardConfig(name, ownerID string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_changeboard" "test" {
  name       = %q
  owner_id   = %q
  owner_type = "organization"

  filter {
    filter_type = "entity_type"
    key         = "type"
    value       = "service"
    operator    = "equal"
  }

  group {
    name  = "service"
    order = "asc"
  }
}
`, name, ownerID)
}

func testAccSensitiveDataConfig(region, name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_sensitive_data_rule" "test" {
  region            = %q
  name              = %q
  telemetry         = "logs"
  order             = 1
  scan_email        = true
  scan_phone_number = false
  scan_credit_card  = false
  action_name       = "redact"
}
`, region, name)
}

func testAccPhysicalIndexConfig(region, name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_physical_index" "test" {
  region    = %q
  name      = %q
  telemetry = "logs"
  retain    = false

  filters {
    key      = "attributes[\"service\"]"
    value    = "tf-test"
    operator = "equals"
  }
}
`, region, name)
}

func testAccCheckRehydrationDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_rehydration" {
			continue
		}
		region, id, parseErr := parseRegionID(rs.Primary.ID)
		if parseErr != nil {
			continue
		}
		if resp, getErr := c.GetOTelRehydration(id, region); getErr == nil {
			if strings.Contains(strings.ToLower(resp.Status), "delete") {
				continue
			}
			return fmt.Errorf("rehydration %s still exists (status=%s)", rs.Primary.ID, resp.Status)
		}
	}
	return nil
}

func testAccRehydrationConfig(region, name string, from, to int64) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_rehydration" "test" {
  region         = %q
  name           = %q
  physical_index = "logs"
  telemetry      = "logs"
  from           = %d
  to             = %d
  message        = "tf acceptance test"
}
`, region, name, from, to)
}

func testAccCheckStreamingAggregationDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_streaming_aggregation" {
			continue
		}
		region, clusterID, id, parseErr := parseOTelResourceID(rs.Primary.ID)
		if parseErr != nil {
			continue
		}
		if _, getErr := c.GetStreamingAggregation(clusterID, region, id); getErr == nil {
			return fmt.Errorf("streaming aggregation %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccStreamingAggregationConfig(region, name, outMetric string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_streaming_aggregation" "test" {
  region        = %q
  name          = %q
  telemetry     = "metrics"
  metric        = "http_requests_total"
  resolution    = "1m"
  aggregation   = "sum"
  clause        = "without"
  labels        = ["instance"]
  output_metric = %q
}
`, region, name, outMetric)
}

func testAccStreamingAggregationConfigUpdated(region, name, outMetric string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_streaming_aggregation" "test" {
  region        = %q
  name          = %q
  telemetry     = "metrics"
  metric        = "http_requests_total"
  resolution    = "5m"
  aggregation   = "max"
  clause        = "without"
  labels        = ["instance"]
  output_metric = %q
}
`, region, name, outMetric)
}

func testAccCheckColdStorageBucketDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_cold_storage_bucket" {
			continue
		}
		region, id, parseErr := parseRegionID(rs.Primary.ID)
		if parseErr != nil {
			continue
		}
		if _, getErr := c.GetColdStorageBucket(id, region); getErr == nil {
			return fmt.Errorf("cold storage bucket %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccColdStorageBucketConfig(region, name, awsRegion, bucket, role string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_cold_storage_bucket" "test" {
  region     = %q
  name       = %q
  aws_region = %q
  aws_bucket = %q
  auth_type  = "role"
  aws_role   = %q
  default    = false
}
`, region, name, awsRegion, bucket, role)
}

func testAccCheckColdStorageBackupDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_cold_storage_backup" {
			continue
		}
		region, id, parseErr := parseRegionID(rs.Primary.ID)
		if parseErr != nil {
			continue
		}
		if _, getErr := c.GetColdStorageBackup(id, region); getErr == nil {
			return fmt.Errorf("cold storage backup %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccColdStorageBackupConfig(region, name, bucketName string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_cold_storage_backup" "test" {
  region      = %q
  name        = %q
  bucket_name = %q
  granularity = "index"
  enabled     = true
}
`, region, name, bucketName)
}

func testAccCheckS3IngestDestroy(s *terraform.State) error {
	c, err := testAccClientFromEnv()
	if err != nil {
		return nil
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "last9_s3_ingest" {
			continue
		}
		region, id, parseErr := parseRegionID(rs.Primary.ID)
		if parseErr != nil {
			continue
		}
		if _, getErr := c.GetS3Ingest(id, region); getErr == nil {
			return fmt.Errorf("s3 ingest %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccS3IngestConfig(region, name, awsRegion, bucket, role string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "last9_s3_ingest" "test" {
  region     = %q
  name       = %q
  aws_region = %q
  aws_bucket = %q
  aws_role   = %q
  auth_type  = "role"
  default    = false
}
`, region, name, awsRegion, bucket, role)
}
