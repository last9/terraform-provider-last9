# Testing Guide - Last9 Terraform Provider

This guide explains how to run tests for the Last9 Terraform Provider, including the new scheduled search alerts and notification destination features.

## Test Types

### 1. Unit Tests
Fast tests that don't require API calls. These test helper functions, data transformations, and validation logic.

### 2. Acceptance Tests
End-to-end tests that create real resources in Last9. These require valid API credentials.

## Prerequisites

### Required Environment Variables

For all tests:
```bash
export LAST9_ORG="your-org-slug"
export LAST9_REFRESH_TOKEN="your-refresh-token"
# OR
export LAST9_API_TOKEN="your-api-token"
```

For acceptance tests (resource-specific):
```bash
# For alert tests
export LAST9_TEST_ENTITY_ID="entity-uuid-here"

# For scheduled search alert tests
export LAST9_TEST_REGION="ap-south-1"  # Optional, defaults to ap-south-1
export LAST9_TEST_NOTIFICATION_DEST_ID="12345"  # Numeric ID

# For notification destination tests
export LAST9_TEST_NOTIFICATION_DEST_NAME="Engineering Slack"
export LAST9_TEST_NOTIFICATION_DEST_ID="12345"
```

### How to Get Test Values

**Entity ID:**
```bash
# List entities in your organization
curl -H "X-LAST9-API-TOKEN: Bearer $LAST9_API_TOKEN" \
  https://api.last9.io/organizations/$LAST9_ORG/entities | jq '.[0].id'
```

**Notification Destination ID and Name:**
```bash
# List notification destinations
curl -H "authorization: Bearer $TOKEN" \
  https://api.last9.io/organizations/$LAST9_ORG/notification_settings \
  | jq '.notification_destinations[] | {id, name, type}'
```

## Running Tests

### Run All Unit Tests

```bash
make test
```

Or directly with Go:
```bash
go test ./internal/provider -run '^Test(Resource|Provider)' -v
```

### Run All Tests (Unit + Acceptance)

```bash
make test-all
```

### Run Only Acceptance Tests

```bash
make testacc
```

Or directly:
```bash
TF_ACC=1 go test ./internal/provider -v -run '^TestAcc'
```

### Run Specific Tests

#### Scheduled Search Alert Tests

**All scheduled search alert tests:**
```bash
TF_ACC=1 go test ./internal/provider -v -run 'TestAccScheduledSearchAlert'
```

**Basic CRUD test:**
```bash
TF_ACC=1 go test ./internal/provider -v -run 'TestAccScheduledSearchAlert_basic'
```

**Update test:**
```bash
TF_ACC=1 go test ./internal/provider -v -run 'TestAccScheduledSearchAlert_update'
```

**Grouping test:**
```bash
TF_ACC=1 go test ./internal/provider -v -run 'TestAccScheduledSearchAlert_withGrouping'
```

**Unit tests for helpers:**
```bash
go test ./internal/provider -v -run 'TestExpandPostProcessors|TestFlattenPostProcessors'
```

#### Notification Destination Tests

**All notification destination tests:**
```bash
TF_ACC=1 go test ./internal/provider -v -run 'TestAccDataSourceNotificationDestination'
```

**Lookup by name:**
```bash
TF_ACC=1 go test ./internal/provider -v -run 'TestAccDataSourceNotificationDestination_byName'
```

**Lookup by ID:**
```bash
TF_ACC=1 go test ./internal/provider -v -run 'TestAccDataSourceNotificationDestination_byID'
```

### Run All New Feature Tests

```bash
# Unit tests
go test ./internal/provider -v -run 'TestExpand|TestFlatten'

# Acceptance tests
TF_ACC=1 go test ./internal/provider -v \
  -run 'TestAccScheduledSearchAlert|TestAccDataSourceNotificationDestination'
```

## Test Coverage

### Scheduled Search Alert Resource

**Unit Tests:**
- ✅ `TestExpandPostProcessors` - Tests post-processor expansion with various scenarios
- ✅ `TestFlattenPostProcessors` - Tests post-processor flattening

**Acceptance Tests:**
- ✅ `TestAccScheduledSearchAlert_basic` - Basic create, read, import
- ✅ `TestAccScheduledSearchAlert_update` - Update threshold and frequency
- ✅ `TestAccScheduledSearchAlert_withGrouping` - Alert with grouped aggregation

### Notification Destination Data Source

**Acceptance Tests:**
- ✅ `TestAccDataSourceNotificationDestination_byName` - Lookup by name
- ✅ `TestAccDataSourceNotificationDestination_byID` - Lookup by ID
- ✅ `TestAccDataSourceNotificationDestination_attributes` - Verify all attributes

## Test Scenarios Covered

### Scheduled Search Alerts

1. **Basic Alert Creation**
   - Simple error count alert
   - Single aggregation function
   - Single notification destination

2. **Alert Updates**
   - Threshold value changes
   - Search frequency changes
   - Maintains resource identity

3. **Grouped Aggregations**
   - Grouping by service/endpoint
   - Multiple filter conditions
   - Complex query structures

4. **Edge Cases (Unit Tests)**
   - Empty post-processor list (should fail)
   - Invalid JSON in function (should fail)
   - Invalid JSON in groupby (should fail)
   - Multiple aggregates
   - Multiple post-processors

### Notification Destinations

1. **Lookup Methods**
   - By name (case-sensitive)
   - By numeric ID

2. **Attribute Verification**
   - All computed fields populated
   - Type, destination, global flags
   - In-use status

## Debugging Failed Tests

### Common Issues

#### "Skipping test - LAST9_TEST_* not set"
**Solution:** Export the required environment variable
```bash
export LAST9_TEST_NOTIFICATION_DEST_ID="12345"
```

#### "notification destination with ID X not found"
**Solution:** Verify the destination exists and ID is correct
```bash
curl -H "authorization: Bearer $TOKEN" \
  https://api.last9.io/organizations/$LAST9_ORG/notification_settings \
  | jq '.notification_destinations[] | select(.id == 12345)'
```

#### "failed to create scheduled search alert"
**Solution:** Check:
1. Query JSON is valid: `echo $QUERY | jq .`
2. Notification destination exists and is active
3. Region is correct for your organization

#### "API error: 401 - Unauthorized"
**Solution:** Refresh your tokens
```bash
# Verify token is valid
curl -H "authorization: Bearer $LAST9_API_TOKEN" \
  https://api.last9.io/organizations/$LAST9_ORG/notification_settings
```

### Verbose Test Output

For more detailed test output:
```bash
TF_LOG=DEBUG TF_ACC=1 go test ./internal/provider -v -run 'TestAccScheduledSearchAlert_basic'
```

## Test Data Cleanup

Acceptance tests should clean up resources automatically. If manual cleanup is needed:

### Delete Scheduled Search Alert
```bash
# This requires fetching all alerts, filtering out the test alert, and re-posting
# Easier to do via Terraform:
terraform destroy
```

### Notification Destinations
Notification destinations are read-only in tests (data source only), so no cleanup needed.

## Continuous Integration

### GitHub Actions (Future)

Tests will run automatically on:
- Pull requests
- Pushes to main branch
- Release tags

CI configuration will be in `.github/workflows/test.yml`

## Best Practices

1. **Always set required environment variables** before running acceptance tests
2. **Run unit tests first** - they're fast and catch most issues
3. **Use specific test names** when debugging to save time
4. **Check API quotas** if tests fail intermittently
5. **Keep test data minimal** to avoid cluttering your Last9 organization
6. **Use test-specific naming** (prefix with "Test" or "TF-Test") for easy identification

## Adding New Tests

When adding new features, follow these patterns:

### Unit Test Template
```go
func TestYourFunction(t *testing.T) {
	tests := []struct {
		name    string
		input   YourInputType
		want    YourOutputType
		wantErr bool
	}{
		{
			name:    "valid case",
			input:   validInput,
			want:    expectedOutput,
			wantErr: false,
		},
		{
			name:    "error case",
			input:   invalidInput,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YourFunction(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
```

### Acceptance Test Template
```go
func TestAccYourResource_scenario(t *testing.T) {
	resourceName := "last9_your_resource.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(),
		CheckDestroy:      testAccCheckYourResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccYourResourceConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckYourResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "field", "value"),
				),
			},
		},
	})
}
```

## Performance

Typical test execution times:

- **Unit tests**: < 1 second
- **Acceptance test (single)**: 5-15 seconds
- **All acceptance tests**: 2-5 minutes
- **Full test suite**: 3-6 minutes

## Support

For test-related issues:
- Check this guide first
- Review test output for specific errors
- Verify environment variables are set correctly
- Ensure API access is working with curl

## Related Documentation

- [Main README](README.md) - Provider overview
- [Authentication Guide](docs/AUTHENTICATION.md) - Auth setup
- [Examples](examples/) - Usage examples
- [Makefile](Makefile) - Available make targets
