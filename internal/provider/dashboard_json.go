package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dashboardJSONSchema(kind, description string) *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeString, Optional: true, Computed: true,
		Description: description + " Use jsonencode(); read/import preserves the native JSON.",
		DiffSuppressFunc: func(_ string, old, next string, _ *schema.ResourceData) bool {
			return jsonStringsEqual(old, next)
		},
		ValidateDiagFunc: func(value interface{}, path cty.Path) diag.Diagnostics {
			if diagnostics := validateOptionalJSONString(value, path); diagnostics.HasError() {
				return diagnostics
			}
			text := strings.TrimSpace(value.(string))
			if text == "" || (kind == "array" && text[0] == '[') || (kind == "object" && text[0] == '{') {
				return nil
			}
			return diag.Diagnostics{diag.Diagnostic{Severity: diag.Error, Summary: fmt.Sprintf("expected JSON %s", kind), AttributePath: path}}
		},
	}
}

func dashboardJSONValue(value interface{}) json.RawMessage {
	text, _ := value.(string)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return json.RawMessage(text)
}

func validateDashboardLogs(panel, visualization map[string]interface{}) error {
	queries := dashboardPanelValues(panel["query"])
	isLogs := visualization["type"] == "logs"
	if !isLogs && len(dashboardJSONValue(visualization["logs_config_json"])) > 0 {
		return fmt.Errorf("logs_config_json is only valid for logs visualization")
	}
	for _, rawQuery := range queries {
		query, ok := rawQuery.(map[string]interface{})
		if !ok {
			return fmt.Errorf("query block must be a known object")
		}
		if !isLogs && query["query_type"] == "log_raw" {
			return fmt.Errorf("log_raw query is only valid for logs visualization")
		}
	}
	if !isLogs {
		return nil
	}
	return validateRawLogsQuery(panel, visualization)
}

func validateDashboardNativeJSONStrings(panel, visualization map[string]interface{}) error {
	if err := validateDashboardValueMappingsJSON(visualization["value_mappings_json"]); err != nil {
		return err
	}
	return validateDashboardFieldOverridesJSON(panel["field_overrides_json"])
}

func validateDashboardValueMappingsJSON(value interface{}) error {
	text, _ := value.(string)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var mappings []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &mappings); err != nil {
		return fmt.Errorf("value_mappings_json must be a JSON array of objects: %w", err)
	}
	for i, mapping := range mappings {
		if raw, ok := mapping["value"]; ok {
			if err := validateJSONStringToken(raw, fmt.Sprintf("value_mappings_json[%d].value", i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDashboardFieldOverridesJSON(value interface{}) error {
	text, _ := value.(string)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var overrides []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &overrides); err != nil {
		return fmt.Errorf("field_overrides_json must be a JSON array of objects: %w", err)
	}
	for overrideIndex, override := range overrides {
		propertiesRaw, ok := override["properties"]
		if !ok {
			continue
		}
		var properties map[string]json.RawMessage
		if err := json.Unmarshal(propertiesRaw, &properties); err != nil {
			continue
		}
		thresholdsRaw, ok := properties["thresholds"]
		if !ok {
			continue
		}
		var thresholds []map[string]json.RawMessage
		if err := json.Unmarshal(thresholdsRaw, &thresholds); err != nil {
			continue
		}
		for thresholdIndex, threshold := range thresholds {
			if raw, ok := threshold["value"]; ok {
				label := fmt.Sprintf("field_overrides_json[%d].properties.thresholds[%d].value", overrideIndex, thresholdIndex)
				if err := validateJSONStringToken(raw, label); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateJSONStringToken(raw json.RawMessage, label string) error {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("%s must be a JSON string", label)
	}
	return nil
}

func validateRawLogsQuery(panel, visualization map[string]interface{}) error {
	queries := dashboardPanelValues(panel["query"])
	if panel["version"].(int) < 1 || len(queries) != 1 {
		return fmt.Errorf("logs visualization requires a V1 panel with exactly one query")
	}
	query, ok := queries[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("logs visualization query must be a known object")
	}
	if query["telemetry"] != "logs" || (query["query_type"] != "log_ql" && query["query_type"] != "log_raw") {
		return fmt.Errorf("logs visualization requires a logs query of type log_ql or log_raw")
	}
	if len(dashboardJSONValue(visualization["logs_config_json"])) == 0 {
		return fmt.Errorf("logs visualization requires logs_config_json")
	}
	return nil
}
