package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceScheduledSearchAlert() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScheduledSearchAlertCreate,
		ReadContext:   resourceScheduledSearchAlertRead,
		UpdateContext: resourceScheduledSearchAlertUpdate,
		DeleteContext: resourceScheduledSearchAlertDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region for the scheduled search alert",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the scheduled search alert",
			},
			"query_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "logjson-aggregate",
				Description:  "Query type (logjson-aggregate, logjson-raw, etc.)",
				ValidateFunc: validation.StringInSlice([]string{"logjson-aggregate", "logjson-raw"}, false),
			},
			"physical_index": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "logs",
				Description: "Physical index to query (logs, traces, metrics)",
			},
			"telemetry": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "logs",
				Description:  "Telemetry type",
				ValidateFunc: validation.StringInSlice([]string{"logs", "traces", "metrics"}, false),
			},
			"query": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Log query as JSON-encoded pipeline array",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					var pipeline []interface{}
					if err := json.Unmarshal([]byte(v), &pipeline); err != nil {
						errs = append(errs, fmt.Errorf("%q must be valid JSON array: %s", key, err))
					}
					return
				},
			},
			"post_processor": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Post-processor configuration for aggregation",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Post-processor type (aggregate)",
						},
						"aggregates": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Aggregation functions",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"function": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Aggregation function as JSON (e.g., {\"$count\": []})",
										ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
											v := val.(string)
											var fn map[string]interface{}
											if err := json.Unmarshal([]byte(v), &fn); err != nil {
												errs = append(errs, fmt.Errorf("%q must be valid JSON object: %s", key, err))
											}
											return
										},
									},
									"as": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Output field name for aggregation result",
									},
								},
							},
						},
						"groupby": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "{}",
							Description: "Group by configuration as JSON object",
							ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
								v := val.(string)
								var gb map[string]interface{}
								if err := json.Unmarshal([]byte(v), &gb); err != nil {
									errs = append(errs, fmt.Errorf("%q must be valid JSON object: %s", key, err))
								}
								return
							},
						},
					},
				},
			},
			"search_frequency": {
				Type:         schema.TypeInt,
				Required:     true,
				Description:  "How often to run the search in seconds (minimum 60, maximum 86400)",
				ValidateFunc: validation.IntBetween(60, 86400), // 1 minute to 24 hours
			},
			"threshold": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Threshold configuration for alerting",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"operator": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "Comparison operator (>, <, >=, <=, ==, !=)",
							ValidateFunc: validation.StringInSlice([]string{">", "<", ">=", "<=", "==", "!="}, false),
						},
						"value": {
							Type:        schema.TypeFloat,
							Required:    true,
							Description: "Threshold value to compare against",
						},
					},
				},
			},
			"alert_destinations": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of notification destination IDs",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
		},
	}
}

func resourceScheduledSearchAlertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	region := d.Get("region").(string)

	// Build the scheduled search alert
	alert, err := buildScheduledSearchAlert(d, apiClient)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to build scheduled search alert: %w", err))
	}

	// Create the alert
	result, err := apiClient.CreateScheduledSearchAlert(region, alert)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create scheduled search alert: %w", err))
	}

	// Set composite ID: region:alert_id:rule_name
	d.SetId(fmt.Sprintf("%s:%s:%s", region, result.ID, result.RuleName))

	return resourceScheduledSearchAlertRead(ctx, d, m)
}

func resourceScheduledSearchAlertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse composite ID
	idParts := strings.Split(d.Id(), ":")
	if len(idParts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid ID format, expected region:response_id:rule_name"))
	}
	region := idParts[0]
	ruleName := idParts[2]

	// Get all scheduled search alerts for the region
	alerts, err := apiClient.GetScheduledSearchAlerts(region)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read scheduled search alerts: %w", err))
	}

	// Find the alert by name
	var alert *client.ScheduledSearchAlertFull
	for i := range alerts {
		if alerts[i].RuleName == ruleName {
			alert = &alerts[i]
			break
		}
	}

	if alert == nil {
		// Alert not found, remove from state
		d.SetId("")
		return nil
	}

	// Set all fields
	d.Set("region", region)
	d.Set("name", alert.RuleName)
	d.Set("query_type", alert.QueryType)
	d.Set("physical_index", alert.PhysicalIndex)
	d.Set("telemetry", alert.Properties.Telemetry)
	d.Set("query", alert.Properties.Query)
	d.Set("search_frequency", alert.Properties.SearchFrequency)

	// Set post_processor
	postProcessors := flattenPostProcessors(alert.Properties.PostProcessor)
	d.Set("post_processor", postProcessors)

	// Set threshold
	threshold := []map[string]interface{}{
		{
			"operator": alert.Properties.Threshold.Operator,
			"value":    alert.Properties.Threshold.Value,
		},
	}
	d.Set("threshold", threshold)

	// Note: We don't update alert_destinations from the API response because
	// the API returns internal association IDs, not the original notification
	// destination IDs that the user specified. We preserve the user's config.

	return nil
}

func resourceScheduledSearchAlertUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse composite ID
	idParts := strings.Split(d.Id(), ":")
	if len(idParts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid ID format, expected region:alert_id:rule_name"))
	}
	region := idParts[0]
	alertID := idParts[1]

	// Build the updated alert
	alert, err := buildScheduledSearchAlert(d, apiClient)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to build scheduled search alert: %w", err))
	}

	// Update the alert using its ID
	result, err := apiClient.UpdateScheduledSearchAlert(region, alertID, alert)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update scheduled search alert: %w", err))
	}

	// Update ID (alert ID may change, name may change)
	d.SetId(fmt.Sprintf("%s:%s:%s", region, result.ID, result.RuleName))

	return resourceScheduledSearchAlertRead(ctx, d, m)
}

func resourceScheduledSearchAlertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)

	// Parse composite ID
	idParts := strings.Split(d.Id(), ":")
	if len(idParts) != 3 {
		return diag.FromErr(fmt.Errorf("invalid ID format, expected region:alert_id:rule_name"))
	}
	region := idParts[0]
	alertID := idParts[1]

	// Delete the alert using its ID
	err := apiClient.DeleteScheduledSearchAlert(region, alertID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete scheduled search alert: %w", err))
	}

	d.SetId("")
	return nil
}

// Helper functions

func buildScheduledSearchAlert(d *schema.ResourceData, apiClient *client.Client) (*client.ScheduledSearchAlert, error) {
	// Get basic fields
	name := d.Get("name").(string)
	queryType := d.Get("query_type").(string)
	physicalIndex := d.Get("physical_index").(string)
	telemetry := d.Get("telemetry").(string)
	query := d.Get("query").(string)
	searchFrequency := d.Get("search_frequency").(int)

	// Build post-processors
	postProcessors, err := expandPostProcessors(d.Get("post_processor").([]interface{}))
	if err != nil {
		return nil, fmt.Errorf("failed to expand post_processor: %w", err)
	}

	// Get threshold
	thresholdList := d.Get("threshold").([]interface{})
	thresholdMap := thresholdList[0].(map[string]interface{})
	threshold := client.Threshold{
		Operator: thresholdMap["operator"].(string),
		Value:    thresholdMap["value"].(float64),
	}

	// Get alert destinations
	destIDs := d.Get("alert_destinations").([]interface{})
	alertDestinations := make([]client.NotificationDestination, len(destIDs))
	for i, idInterface := range destIDs {
		id := idInterface.(int)
		dest, err := apiClient.GetNotificationDestination(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get notification destination %d: %w", id, err)
		}
		alertDestinations[i] = *dest
	}

	// Build the alert
	alert := &client.ScheduledSearchAlert{
		RuleName:      name,
		QueryType:     queryType,
		PhysicalIndex: physicalIndex,
		RuleType:      "scheduled_search",
		Properties: client.ScheduledSearchProperties{
			Telemetry:         telemetry,
			Query:             query,
			PostProcessor:     postProcessors,
			SearchFrequency:   searchFrequency,
			AlertDestinations: alertDestinations,
			Threshold:         threshold,
		},
	}

	return alert, nil
}

func expandPostProcessors(raw []interface{}) ([]client.PostProcessor, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("at least one post_processor is required")
	}

	processors := make([]client.PostProcessor, len(raw))
	for i, r := range raw {
		m := r.(map[string]interface{})

		// Get aggregates
		aggregatesRaw := m["aggregates"].([]interface{})
		aggregates := make([]client.Aggregate, len(aggregatesRaw))
		for j, aggRaw := range aggregatesRaw {
			aggMap := aggRaw.(map[string]interface{})

			// Parse function JSON
			functionStr := aggMap["function"].(string)
			var function map[string]interface{}
			if err := json.Unmarshal([]byte(functionStr), &function); err != nil {
				return nil, fmt.Errorf("failed to parse aggregate function: %w", err)
			}

			aggregates[j] = client.Aggregate{
				Function: function,
				As:       aggMap["as"].(string),
			}
		}

		// Parse groupby JSON
		groupbyStr := m["groupby"].(string)
		var groupby map[string]interface{}
		if err := json.Unmarshal([]byte(groupbyStr), &groupby); err != nil {
			return nil, fmt.Errorf("failed to parse groupby: %w", err)
		}

		processors[i] = client.PostProcessor{
			Type:       m["type"].(string),
			Aggregates: aggregates,
			Groupby:    groupby,
		}
	}

	return processors, nil
}

func flattenPostProcessors(processors []client.PostProcessor) []interface{} {
	if len(processors) == 0 {
		return []interface{}{}
	}

	result := make([]interface{}, len(processors))
	for i, proc := range processors {
		// Flatten aggregates
		aggregates := make([]interface{}, len(proc.Aggregates))
		for j, agg := range proc.Aggregates {
			functionJSON, _ := json.Marshal(agg.Function)
			aggregates[j] = map[string]interface{}{
				"function": string(functionJSON),
				"as":       agg.As,
			}
		}

		// Flatten groupby
		groupbyJSON, _ := json.Marshal(proc.Groupby)

		result[i] = map[string]interface{}{
			"type":       proc.Type,
			"aggregates": aggregates,
			"groupby":    string(groupbyJSON),
		}
	}

	return result
}
