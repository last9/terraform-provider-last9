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
			return diag.Diagnostics{{Severity: diag.Error, Summary: fmt.Sprintf("expected JSON %s", kind), AttributePath: path}}
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
