package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func resourceAlert() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAlertCreate,
		ReadContext:   resourceAlertRead,
		UpdateContext: resourceAlertUpdate,
		DeleteContext: resourceAlertDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
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
			"indicator": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Indicator name",
			},
			"expression": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alert expression (for dynamic alerts)",
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
				Description:  "Alert severity (breach, threat, info)",
				ValidateFunc: validation.StringInSlice([]string{"breach", "threat", "info"}, false),
			},
			"mute": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether alert is muted",
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
		},
	}
}

func resourceAlertCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)

	req := &client.AlertCreateRequest{
		RuleName:                     d.Get("name").(string),
		PrimaryIndicator:             d.Get("indicator").(string),
		Severity:                     d.Get("severity").(string),
		IsDisabled:                   d.Get("is_disabled").(bool),
		GroupTimeseriesNotifications: d.Get("group_timeseries_notifications").(bool),
		ExpressionArgs:               make(map[string]interface{}),
	}

	// Handle mute
	if d.Get("mute").(bool) {
		req.MuteUntil = -1 // mute forever
	} else {
		req.MuteUntil = 0 // unmute
	}

	// Handle expression-based alerts
	if expression, ok := d.GetOk("expression"); ok {
		req.Expression = expression.(string)
	} else {
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
		return diag.FromErr(fmt.Errorf("failed to create alert: %w", err))
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
	d.Set("mute", alert.MuteUntil == -1)
	d.Set("is_disabled", alert.IsDisabled)
	d.Set("group_timeseries_notifications", alert.GroupTimeseriesNotifications)

	// Parse condition for static alerts to extract threshold values
	if alert.Condition != "" && alert.EvalWindow > 0 {
		// Parse condition like "expr > 100" or "expr < 50"
		// Extract threshold and operator
		if err := parseAndSetCondition(d, alert.Condition, alert.EvalWindow, alert.AlertCondition); err != nil {
			// Log warning but don't fail - condition parsing is best effort
			// The alert will still work, just won't have threshold values in state
		}
	}

	// Set properties
	props := []interface{}{
		map[string]interface{}{
			"description": alert.Properties.Description,
			"runbook_url": func() string {
				if alert.Properties.Runbook != nil {
					if runbook, ok := alert.Properties.Runbook["link"].(string); ok {
						return runbook
					}
				}
				return ""
			}(),
			"annotations": alert.Properties.Annotations,
		},
	}
	d.Set("properties", props)

	return nil
}

func resourceAlertUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)

	req := &client.AlertUpdateRequest{}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		req.RuleName = &name
	}
	if d.HasChange("indicator") {
		indicator := d.Get("indicator").(string)
		req.PrimaryIndicator = &indicator
	}
	if d.HasChange("expression") {
		expression := d.Get("expression").(string)
		req.Expression = &expression
	}
	if d.HasChange("severity") {
		severity := d.Get("severity").(string)
		req.Severity = &severity
	}
	if d.HasChange("mute") {
		mute := d.Get("mute").(bool)
		if mute {
			muteUntil := -1
			req.MuteUntil = &muteUntil
		} else {
			muteUntil := 0
			req.MuteUntil = &muteUntil
		}
	}
	if d.HasChange("is_disabled") {
		isDisabled := d.Get("is_disabled").(bool)
		req.IsDisabled = &isDisabled
	}
	if d.HasChange("group_timeseries_notifications") {
		group := d.Get("group_timeseries_notifications").(bool)
		req.GroupTimeseriesNotifications = &group
	}

	// Handle static threshold changes
	if d.HasChange("greater_than") || d.HasChange("less_than") || d.HasChange("bad_minutes") || d.HasChange("total_minutes") {
		if greaterThan, ok := d.GetOk("greater_than"); ok {
			badMinutes := d.Get("bad_minutes").(int)
			totalMinutes := d.Get("total_minutes").(int)
			condition := fmt.Sprintf("expr > %f", greaterThan.(float64))
			alertCondition := fmt.Sprintf("count_true(result) >= %d", badMinutes)
			req.Condition = &condition
			req.AlertCondition = &alertCondition
			req.EvalWindow = &totalMinutes
		} else if lessThan, ok := d.GetOk("less_than"); ok {
			badMinutes := d.Get("bad_minutes").(int)
			totalMinutes := d.Get("total_minutes").(int)
			condition := fmt.Sprintf("expr < %f", lessThan.(float64))
			alertCondition := fmt.Sprintf("count_true(result) >= %d", badMinutes)
			req.Condition = &condition
			req.AlertCondition = &alertCondition
			req.EvalWindow = &totalMinutes
		}
	}

	if d.HasChange("properties") {
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
	}

	_, err := apiClient.UpdateAlert(entityID, d.Id(), req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update alert: %w", err))
	}

	return resourceAlertRead(ctx, d, m)
}

func resourceAlertDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	apiClient := m.(*client.Client)
	entityID := d.Get("entity_id").(string)

	err := apiClient.DeleteAlert(entityID, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete alert: %w", err))
	}

	d.SetId("")
	return nil
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
