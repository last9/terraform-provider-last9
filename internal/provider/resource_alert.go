package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// generateKPIName creates a unique KPI name from the rule name plus a random token
func generateKPIName(ruleName string) string {
	token := make([]byte, 4)
	rand.Read(token)
	return fmt.Sprintf("%s-%s", ruleName, hex.EncodeToString(token))
}

func resourceAlert() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAlertCreate,
		ReadContext:   resourceAlertRead,
		UpdateContext: resourceAlertUpdate,
		DeleteContext: resourceAlertDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceAlertImportState,
		},
		Schema: map[string]*schema.Schema{
			"entity_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Entity ID this alert belongs to",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Alert name",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alert description",
			},
			"query": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "PromQL query for the KPI/indicator that will be created for this alert",
			},
			"kpi_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the automatically created KPI for this alert",
			},
			"kpi_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the automatically created KPI for this alert",
			},
			"indicator": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Indicator name (derived from KPI name)",
			},
			"expression": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Alert expression (computed from KPI name)",
			},
			"greater_than": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Threshold value for greater than condition",
			},
			"less_than": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Description: "Threshold value for less than condition",
			},
			"bad_minutes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Minutes threshold must be exceeded",
			},
			"total_minutes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Evaluation window in minutes",
			},
			"severity": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "breach",
				Description:  "Alert severity (breach or threat)",
				ValidateFunc: validation.StringInSlice([]string{"breach", "threat"}, false),
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether alert is disabled",
			},
			"properties": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Alert properties",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"runbook_url": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Runbook URL",
						},
						"annotations": {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Alert annotations",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"group_timeseries_notifications": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Group timeseries notifications",
			},
			"notification_channels": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Notification channel IDs or names to send alerts to",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAlertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)
	ruleName := d.Get("name").(string)
	query := d.Get("query").(string)

	// Step 1: Create a KPI for this alert
	kpiName := generateKPIName(ruleName)
	kpiReq := &client.KPICreateRequest{
		Name: kpiName,
		Definition: client.KPIDefinition{
			Query:  query,
			Source: "levitate",
			Unit:   "count",
		},
		KPIType: "custom",
	}

	kpi, err := apiClient.CreateKPI(entityID, kpiReq)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create KPI for alert: %w", err))
	}

	// Store KPI info in state
	d.Set("kpi_id", kpi.ID)
	d.Set("kpi_name", kpi.Name)
	d.Set("indicator", kpi.Name)

	// Step 2: Create the alert using the KPI
	req := &client.AlertCreateRequest{
		RuleName:                     ruleName,
		PrimaryIndicator:             kpi.Name,
		Severity:                     d.Get("severity").(string),
		IsDisabled:                   d.Get("is_disabled").(bool),
		GroupTimeseriesNotifications: d.Get("group_timeseries_notifications").(bool),
		ExpressionArgs: map[string]interface{}{
			kpi.Name: map[string]interface{}{
				"id": kpi.ID,
			},
		},
	}

	// Handle notification channels
	if v, ok := d.GetOk("notification_channels"); ok {
		channelsList := v.([]interface{})
		channels := make([]string, len(channelsList))
		for i, ch := range channelsList {
			channels[i] = ch.(string)
		}
		req.NotificationChannels = channels
	}

	// Handle static threshold alerts
	if greaterThan, ok := d.GetOk("greater_than"); ok {
		badMinutes := d.Get("bad_minutes").(int)
		totalMinutes := d.Get("total_minutes").(int)
		req.Condition = fmt.Sprintf("expr > %f", greaterThan.(float64))
		req.AlertCondition = fmt.Sprintf("count_true(result) >= %d", badMinutes)
		req.EvalWindow = totalMinutes
	} else if lessThan, ok := d.GetOk("less_than"); ok {
		badMinutes := d.Get("bad_minutes").(int)
		totalMinutes := d.Get("total_minutes").(int)
		req.Condition = fmt.Sprintf("expr < %f", lessThan.(float64))
		req.AlertCondition = fmt.Sprintf("count_true(result) >= %d", badMinutes)
		req.EvalWindow = totalMinutes
	}

	// Handle properties
	if v, ok := d.GetOk("properties"); ok {
		propsList := v.([]interface{})
		if len(propsList) > 0 {
			propsMap := propsList[0].(map[string]interface{})
			req.Properties = client.AlertProperties{
				Description: d.Get("description").(string),
			}
			if runbookURL, ok := propsMap["runbook_url"].(string); ok && runbookURL != "" {
				req.Properties.Runbook = map[string]interface{}{
					"link": runbookURL,
				}
			}
			if annotations, ok := propsMap["annotations"].(map[string]interface{}); ok {
				req.Properties.Annotations = make(map[string]string)
				for k, v := range annotations {
					req.Properties.Annotations[k] = v.(string)
				}
			}
		}
	} else {
		req.Properties = client.AlertProperties{
			Description: d.Get("description").(string),
		}
	}

	alert, err := apiClient.CreateAlert(entityID, req)
	if err != nil {
		// Clean up the KPI if alert creation fails
		apiClient.DeleteKPI(entityID, kpi.ID)
		return diag.FromErr(fmt.Errorf("failed to create alert: %w", err))
	}

	// Validate that we got a valid alert ID back from the API
	if alert.ID == "" {
		// Clean up the KPI since alert creation didn't return a valid ID
		apiClient.DeleteKPI(entityID, kpi.ID)
		return diag.FromErr(fmt.Errorf("alert creation succeeded but API returned empty alert ID - alert may not have been created"))
	}

	d.SetId(alert.ID)
	return resourceAlertRead(ctx, d, m)
}

func resourceAlertRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)

	alert, err := apiClient.GetAlert(entityID, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read alert: %w", err))
	}

	d.Set("entity_id", entityID)
	d.Set("name", alert.Name)
	d.Set("description", alert.Properties.Description)
	d.Set("indicator", alert.Indicator)
	d.Set("expression", alert.Expression)
	d.Set("severity", alert.Severity)
	// Note: mute and is_disabled fields are intentionally not read from API
	// The API may return different values than what was sent, causing drift
	d.Set("group_timeseries_notifications", alert.GroupTimeseriesNotifications)
	d.Set("notification_channels", alert.NotificationChannels)

	// Parse condition for static alerts to extract threshold values
	if alert.Condition != "" && alert.EvalWindow > 0 {
		// Parse condition like "expr > 100" or "expr < 50"
		// Extract threshold and operator
		if err := parseAndSetCondition(d, alert.Condition, alert.EvalWindow, alert.AlertCondition); err != nil {
			// Log warning but don't fail - condition parsing is best effort
			// The alert will still work, just won't have threshold values in state
		}
	}

	// Set properties (only if there are runbook_url or annotations)
	runbookURL := ""
	if alert.Properties.Runbook != nil {
		if link, ok := alert.Properties.Runbook["link"].(string); ok {
			runbookURL = link
		}
	}

	// Only set properties block if there are values to set
	if runbookURL != "" || len(alert.Properties.Annotations) > 0 {
		props := []interface{}{
			map[string]interface{}{
				"runbook_url": runbookURL,
				"annotations": alert.Properties.Annotations,
			},
		}
		d.Set("properties", props)
	}

	return nil
}

func resourceAlertUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)
	oldKPIID := d.Get("kpi_id").(string)
	oldKPIName := d.Get("kpi_name").(string)

	// Track if we need to create a new KPI (name or query changed)
	needsNewKPI := d.HasChange("name") || d.HasChange("query")
	var newKPI *client.KPI

	if needsNewKPI {
		// Create new KPI with new name
		newName := d.Get("name").(string)
		newQuery := d.Get("query").(string)
		newKPIName := generateKPIName(newName)

		kpiReq := &client.KPICreateRequest{
			Name: newKPIName,
			Definition: client.KPIDefinition{
				Query:  newQuery,
				Source: "levitate",
				Unit:   "count",
			},
			KPIType: "custom",
		}

		var err error
		newKPI, err = apiClient.CreateKPI(entityID, kpiReq)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create new KPI for alert update: %w", err))
		}
	}

	req := &client.AlertUpdateRequest{}

	// PUT requires all fields - always include name, severity, indicator refs
	// (Partial updates are not supported for alert-rules per OpenAPI spec)
	name := d.Get("name").(string)
	req.RuleName = &name

	severity := d.Get("severity").(string)
	req.Severity = &severity

	// Determine KPI name and ID to use
	kpiName := d.Get("kpi_name").(string)
	kpiID := d.Get("kpi_id").(string)

	// If we created a new KPI, use its details
	if newKPI != nil {
		kpiName = newKPI.Name
		kpiID = newKPI.ID
	}

	req.PrimaryIndicator = &kpiName
	req.ExpressionArgs = map[string]interface{}{
		kpiName: map[string]interface{}{
			"id": kpiID,
		},
	}

	// Note: mute field is not sent to API - it's ignored by Terraform

	isDisabled := d.Get("is_disabled").(bool)
	req.IsDisabled = &isDisabled

	group := d.Get("group_timeseries_notifications").(bool)
	req.GroupTimeseriesNotifications = &group
	// Always include notification_channels (PUT requires all fields)
	channelsList := d.Get("notification_channels").([]interface{})
	channels := make([]string, len(channelsList))
	for i, ch := range channelsList {
		channels[i] = ch.(string)
	}
	req.NotificationChannels = channels

	// PUT requires all fields - always include condition, alert_condition, eval_window
	// (Partial updates are not supported for alert-rules per OpenAPI spec)
	badMinutes := d.Get("bad_minutes").(int)
	totalMinutes := d.Get("total_minutes").(int)
	alertCondition := fmt.Sprintf("count_true(result) >= %d", badMinutes)
	req.AlertCondition = &alertCondition
	req.EvalWindow = &totalMinutes

	if greaterThan, ok := d.GetOk("greater_than"); ok {
		condition := fmt.Sprintf("expr > %f", greaterThan.(float64))
		req.Condition = &condition
	} else if lessThan, ok := d.GetOk("less_than"); ok {
		condition := fmt.Sprintf("expr < %f", lessThan.(float64))
		req.Condition = &condition
	}

	// Always include properties (PUT requires all fields)
	props := client.AlertProperties{
		Description: d.Get("description").(string),
	}
	if v, ok := d.GetOk("properties"); ok {
		propsList := v.([]interface{})
		if len(propsList) > 0 {
			propsMap := propsList[0].(map[string]interface{})
			if runbookURL, ok := propsMap["runbook_url"].(string); ok && runbookURL != "" {
				props.Runbook = map[string]interface{}{
					"link": runbookURL,
				}
			}
			if annotations, ok := propsMap["annotations"].(map[string]interface{}); ok {
				props.Annotations = make(map[string]string)
				for k, v := range annotations {
					props.Annotations[k] = v.(string)
				}
			}
		}
	}
	req.Properties = &props

	updatedAlert, err := apiClient.UpdateAlert(entityID, d.Id(), req)
	if err != nil {
		// If we created a new KPI but alert update failed, clean it up
		if newKPI != nil {
			apiClient.DeleteKPI(entityID, newKPI.ID)
		}
		return diag.FromErr(fmt.Errorf("failed to update alert: %w", err))
	}

	// Validate that we got a valid alert ID back from the API
	if updatedAlert.ID == "" {
		// If we created a new KPI but got no alert ID, clean it up
		if newKPI != nil {
			apiClient.DeleteKPI(entityID, newKPI.ID)
		}
		return diag.FromErr(fmt.Errorf("alert update succeeded but API returned empty alert ID - alert may not have been updated"))
	}

	// API deletes and recreates alert on update - update the ID
	// (per OpenAPI spec: "Partial updating is not supported for alert-rules.
	// The existing rule is deleted and a new one is created as process of updating alert-rules.")
	if updatedAlert.ID != d.Id() {
		d.SetId(updatedAlert.ID)
	}

	// If we created a new KPI, delete the old one and update state
	if newKPI != nil {
		// Delete old KPI (ignore errors - it may already be gone)
		if oldKPIID != "" && oldKPIName != "" {
			apiClient.DeleteKPI(entityID, oldKPIID)
		}

		// Update state with new KPI info
		d.Set("kpi_id", newKPI.ID)
		d.Set("kpi_name", newKPI.Name)
		d.Set("indicator", newKPI.Name)
	}

	return resourceAlertRead(ctx, d, m)
}

func resourceAlertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)
	kpiID := d.Get("kpi_id").(string)

	// Delete the alert first
	err := apiClient.DeleteAlert(entityID, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete alert: %w", err))
	}

	// Delete the associated KPI (ignore errors - it may already be gone)
	if kpiID != "" {
		apiClient.DeleteKPI(entityID, kpiID)
	}

	d.SetId("")
	return nil
}

// resourceAlertImportState handles importing alerts using composite ID format: entity_id:alert_id
func resourceAlertImportState(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), ":")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return nil, fmt.Errorf("invalid import ID format, expected 'entity_id:alert_id', got: %s", d.Id())
	}

	entityID := idParts[0]
	alertID := idParts[1]

	d.Set("entity_id", entityID)
	d.SetId(alertID)

	return []*schema.ResourceData{d}, nil
}

// parseAndSetCondition parses the alert condition string and sets the appropriate
// schema fields (greater_than, less_than, bad_minutes, total_minutes)
func parseAndSetCondition(d *schema.ResourceData, condition string, evalWindow int, alertCondition string) error {
	// Parse condition like "expr > 100" or "expr < 50"
	// Extract operator and threshold value
	var threshold float64

	if len(condition) > 5 && condition[:5] == "expr " {
		rest := condition[5:]
		if len(rest) > 2 && rest[:2] == "> " {
			if _, err := fmt.Sscanf(rest[2:], "%f", &threshold); err != nil {
				return fmt.Errorf("failed to parse threshold from condition: %w", err)
			}
			d.Set("greater_than", threshold)
		} else if len(rest) > 2 && rest[:2] == "< " {
			if _, err := fmt.Sscanf(rest[2:], "%f", &threshold); err != nil {
				return fmt.Errorf("failed to parse threshold from condition: %w", err)
			}
			d.Set("less_than", threshold)
		}
	}

	// Parse eval_window (total_minutes)
	if evalWindow > 0 {
		d.Set("total_minutes", evalWindow)
	}

	// Parse alert_condition like "count_true(result) >= 5" to extract bad_minutes
	if alertCondition != "" {
		var badMinutes int
		if _, err := fmt.Sscanf(alertCondition, "count_true(result) >= %d", &badMinutes); err == nil {
			d.Set("bad_minutes", badMinutes)
		}
	}

	return nil
}
