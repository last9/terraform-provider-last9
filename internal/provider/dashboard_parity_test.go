package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func parityDashboardData(t *testing.T) *schema.ResourceData {
	t.Helper()
	return schema.TestResourceDataRaw(t, resourceDashboard().Schema, map[string]interface{}{
		"region": "us-east-1", "name": "Gateway", "links_json": `[{"title":"Docs","url":"https://example.test"}]`,
		"variable": []interface{}{map[string]interface{}{
			"display_name": "Environment", "target": "env", "type": "label", "source": "env",
			"regex": "prod.*", "include_all": true, "all_value": ".*", "multiple": true,
		}},
		"panel": []interface{}{map[string]interface{}{
			"name": "Logs", "version": 1,
			"links_json":           `[{"title":"Docs","url":"https://example.test"}]`,
			"data_links_json":      `[{"title":"Trace","url":"https://example.test/${__data.fields.trace_id}"}]`,
			"field_overrides_json": `[{"matcher":{"type":"regex","value":"status.*"},"properties":{"hidden":true,"thresholds":[{"value":500,"color":"red","colorTarget":"background"}],"data_links":[{"title":"Logs","url":"https://example.test/${__value.raw}"}]}}]`,
			"transformations_json": `[{"id":"organize","options":{"excludeByName":{"Time":true}}}]`,
			"layout":               []interface{}{map[string]interface{}{"x": 0, "y": 0, "w": 24, "h": 8}},
			"visualization": []interface{}{map[string]interface{}{
				"type": "logs", "logs_config_json": `{"columns":["timestamp","body","service","severity"],"sort_order":"desc","row_limit":1000,"severity_coloring":false}`,
				"value_mappings_json": `[{"type":"value","value":"500","text":"Error"}]`,
			}},
			"query": []interface{}{map[string]interface{}{"name": "A", "expr": `{env=~"$env"}`, "telemetry": "logs", "query_type": "log_ql"}},
		}, map[string]interface{}{"name": "Details", "collapsed": true, "visualization": []interface{}{map[string]interface{}{"type": "section"}}}},
	})
}

func parityAPI(t *testing.T, stored *map[string]interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := validateParityAPIValueMappings(body); err != nil {
				t.Errorf("API rejected invalid value_mappings shape: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := json.NewDecoder(bytes.NewReader(body)).Decode(stored); err != nil {
				t.Error(err)
			}
			dashboard := (*stored)["dashboard"].(map[string]interface{})
			dashboard["id"] = "dashboard-1"
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(*stored); err != nil {
			t.Error(err)
		}
	}
}

func validateParityAPIValueMappings(body []byte) error {
	var payload struct {
		Dashboard struct {
			Panels []struct {
				Visualization struct {
					ValueMappings []struct {
						Type  string `json:"type"`
						Value string `json:"value"`
						Text  string `json:"text"`
					} `json:"value_mappings"`
				} `json:"visualization"`
			} `json:"panels"`
		} `json:"dashboard"`
	}
	return json.Unmarshal(body, &payload)
}

func assertParityRequest(t *testing.T, data *schema.ResourceData) {
	t.Helper()
	encoded, err := json.Marshal(buildDashboardRequest(data))
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]interface{}
	if err := json.Unmarshal(encoded, &request); err != nil {
		t.Fatal(err)
	}
	assertParityNativeFields(t, request["dashboard"].(map[string]interface{}))
	initial, _ := json.Marshal(buildDashboardRequest(parityDashboardData(t)))
	var expected map[string]interface{}
	if err := json.Unmarshal(initial, &expected); err != nil {
		t.Fatal(err)
	}
	delete(request["dashboard"].(map[string]interface{}), "id")
	want, _ := json.Marshal(expected)
	got, _ := json.Marshal(request)
	if string(got) != string(want) {
		t.Fatalf("native fields changed across read/import:\ngot %s\nwant %s", got, want)
	}
}

func assertParityNativeFields(t *testing.T, dashboard map[string]interface{}) {
	t.Helper()
	initial := parityDashboardData(t)
	panel := dashboard["panels"].([]interface{})[0].(map[string]interface{})
	source := initial.Get("panel").([]interface{})[0].(map[string]interface{})
	for _, key := range []string{"field_overrides", "transformations", "links", "data_links"} {
		encoded, _ := json.Marshal(panel[key])
		if !jsonStringsEqual(string(encoded), source[key+"_json"].(string)) {
			t.Fatalf("%s was lost or changed: %s", key, encoded)
		}
	}
	visualization := panel["visualization"].(map[string]interface{})
	sourceVisualization := source["visualization"].([]interface{})[0].(map[string]interface{})
	for _, key := range []string{"logs_config", "value_mappings"} {
		encoded, _ := json.Marshal(visualization[key])
		if !jsonStringsEqual(string(encoded), sourceVisualization[key+"_json"].(string)) {
			t.Fatalf("%s was lost or changed: %s", key, encoded)
		}
	}
	variable := dashboard["variables"].([]interface{})[0].(map[string]interface{})
	if variable["regex"] != "prod.*" || variable["include_all"] != true || variable["all_value"] != ".*" {
		t.Fatalf("variable settings lost: %v", variable)
	}
	if dashboard["panels"].([]interface{})[1].(map[string]interface{})["collapsed"] != true {
		t.Fatal("section collapse was lost")
	}
}

func TestDashboardParityCRUDAndImport(t *testing.T) {
	var stored map[string]interface{}
	api := reviewClient(t, parityAPI(t, &stored))
	data := parityDashboardData(t)
	if err := validateDashboardData(data); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []schema.CreateContextFunc{resourceDashboardCreate, resourceDashboardUpdate} {
		if diagnostics := operation(context.Background(), data, api); diagnostics.HasError() {
			t.Fatal(diagnostics)
		}
		assertParityRequest(t, data)
	}
	imported := schema.TestResourceDataRaw(t, resourceDashboard().Schema, nil)
	imported.SetId("us-east-1:dashboard-1")
	if _, err := resourceDashboardImportState(context.Background(), imported, api); err != nil {
		t.Fatal(err)
	}
	if diagnostics := resourceDashboardRead(context.Background(), imported, api); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	assertParityRequest(t, imported)
	if diagnostics := resourceDashboardDelete(context.Background(), data, api); diagnostics.HasError() || data.Id() != "" {
		t.Fatalf("delete diagnostics %v, id %q", diagnostics, data.Id())
	}
}

func TestDashboardParityJSONValidation(t *testing.T) {
	for _, kind := range []string{"array", "object"} {
		validator := dashboardJSONSchema(kind, "Synthetic").ValidateDiagFunc
		for _, value := range []string{"null", `"text"`, "42", "[invalid"} {
			if diagnostics := validator(value, nil); !diagnostics.HasError() {
				t.Fatalf("%s accepted %s", kind, value)
			}
		}
	}
}

func TestDashboardParityAPIRequiresStringValueMappingValue(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "string value", value: `"500"`},
		{name: "numeric value", value: `500`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"dashboard":{"panels":[{"visualization":{"value_mappings":[{"type":"value","value":` + tc.value + `,"text":"Error"}]}}]}}`)
			err := validateParityAPIValueMappings(body)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateParityAPIValueMappings error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestDashboardParityLogsValidation(t *testing.T) {
	for _, mutate := range []func(map[string]interface{}, map[string]interface{}){
		func(panel, visualization map[string]interface{}) { panel["version"] = 0 },
		func(panel, visualization map[string]interface{}) { panel["query"] = []interface{}{} },
		func(panel, visualization map[string]interface{}) { visualization["logs_config_json"] = "" },
		func(panel, visualization map[string]interface{}) { visualization["type"] = "table" },
		func(panel, visualization map[string]interface{}) {
			panel["query"].([]interface{})[0].(map[string]interface{})["query_type"] = "log_json"
		},
	} {
		data := parityDashboardData(t)
		panel := data.Get("panel").([]interface{})[0].(map[string]interface{})
		visualization := panel["visualization"].([]interface{})[0].(map[string]interface{})
		mutate(panel, visualization)
		if err := validateDashboardLogs(panel, visualization); err == nil {
			t.Fatalf("invalid logs configuration accepted: %v", visualization)
		}
	}
}
