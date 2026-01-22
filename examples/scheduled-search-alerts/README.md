# Scheduled Search Alerts Example

This example demonstrates how to create log-based scheduled search alerts with the Last9 Terraform Provider.

## Features Demonstrated

1. **Data Source Usage**: Looking up existing notification destinations by name
2. **Simple Error Count Alert**: Alerting when error count exceeds a threshold
3. **Grouped Aggregation**: Counting errors grouped by endpoint
4. **Frequency Variation**: Different check intervals (1min, 2min, 5min, 10min)
5. **Multiple Operators**: Using `>`, `>=`, `<` operators
6. **Multiple Destinations**: Sending alerts to Slack and PagerDuty

## Prerequisites

1. Last9 organization with logs configured
2. Notification destinations created in Last9 (Slack, PagerDuty, etc.)
3. Environment variables set:
   - `LAST9_REFRESH_TOKEN` or `LAST9_API_TOKEN`
   - `LAST9_ORG`

## Query Structure

The `query` field uses Last9's LogJSON query language, encoded as JSON:

```hcl
query = jsonencode([
  {
    type  = "filter"
    query = {
      "$and" = [
        { "$eq" = ["SeverityText", "ERROR"] }
      ]
    }
  }
])
```

## Post-Processor Configuration

The `post_processor` defines how to aggregate logs:

```hcl
post_processor {
  type = "aggregate"

  aggregates {
    function = jsonencode({ "$count" = [] })
    as       = "error_count"
  }

  groupby = "{}"  # No grouping, or use JSON like {"endpoint": ["attributes.endpoint"]}
}
```

## Threshold Configuration

Define when to trigger an alert:

```hcl
threshold {
  operator = ">"    # Operators: >, <, >=, <=, ==, !=
  value    = 100    # Numeric threshold
}
```

## Usage

1. Update the notification destination names to match your setup:
   ```hcl
   data "last9_notification_destination" "slack_alerts" {
     name = "Your Slack Channel Name"
   }
   ```

2. Initialize Terraform:
   ```bash
   terraform init
   ```

3. Plan the changes:
   ```bash
   terraform plan
   ```

4. Apply the configuration:
   ```bash
   terraform apply
   ```

## Common Query Patterns

### Filter by Severity
```hcl
query = jsonencode([
  {
    type  = "filter"
    query = {
      "$and" = [
        { "$eq" = ["SeverityText", "ERROR"] }
      ]
    }
  }
])
```

### Filter by Service and Severity
```hcl
query = jsonencode([
  {
    type  = "filter"
    query = {
      "$and" = [
        { "$eq" = ["SeverityText", "ERROR"] },
        { "$eq" = ["attributes.service", "api-service"] }
      ]
    }
  }
])
```

### Multiple Conditions with OR
```hcl
query = jsonencode([
  {
    type  = "filter"
    query = {
      "$or" = [
        { "$eq" = ["SeverityText", "ERROR"] },
        { "$eq" = ["SeverityText", "CRITICAL"] }
      ]
    }
  }
])
```

## Aggregation Functions

### Count
```hcl
function = jsonencode({ "$count" = [] })
```

### Sum
```hcl
function = jsonencode({ "$sum" = ["attributes.bytes"] })
```

### Average
```hcl
function = jsonencode({ "$avg" = ["attributes.duration"] })
```

### Max/Min
```hcl
function = jsonencode({ "$max" = ["attributes.value"] })
```

## Grouping Examples

### No Grouping
```hcl
groupby = "{}"
```

### Group by Single Field
```hcl
groupby = jsonencode({
  "service" = ["attributes.service"]
})
```

### Group by Multiple Fields
```hcl
groupby = jsonencode({
  "service"  = ["attributes.service"],
  "endpoint" = ["attributes.endpoint"]
})
```

## Search Frequency Guidelines

- **60 seconds**: For critical alerts that need immediate detection
- **300 seconds (5 min)**: Standard error monitoring
- **600 seconds (10 min)**: Less critical monitoring, activity checks
- **1800 seconds (30 min)**: Periodic checks, trend monitoring

## Notification Destinations

Supported destination types:
- `email`: Email addresses
- `slack`: Slack channels/webhooks
- `pagerduty`: PagerDuty integration
- `webhook`: Custom webhook URLs
- `opsgenie`: Opsgenie alerts
- `msteams`: Microsoft Teams

## Troubleshooting

### Query Syntax Errors
- Ensure JSON is valid in `query` and `function` fields
- Use `jsonencode()` for proper escaping
- Test queries in Last9 UI before adding to Terraform

### Notification Destination Not Found
- Verify the destination exists in Last9
- Check the exact name matches (case-sensitive)
- Use `terraform console` to test data source queries

### Alert Not Triggering
- Check `search_frequency` and `threshold` values
- Verify query returns results in Last9 UI
- Ensure notification destinations are active

## Additional Resources

- [Last9 Documentation](https://docs.last9.io)
- [Terraform Provider Documentation](../README.md)
