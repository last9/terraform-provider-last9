package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestScheduledSearchAlertImportedPipelineSurvivesUnrelatedUpdate(t *testing.T) {
	const resultant = `[{"type":"filter","query":{"$eq":["service","api"]}},{"type":"aggregate","aggregates":[{"function":{"$count":[]},"as":"result"}],"groupby":{}}]`
	var stored map[string]json.RawMessage
	var writes int
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			wantPath := "/logs_settings/scheduled_search"
			if r.Method == http.MethodPut {
				wantPath += "/search-1"
			}
			if !strings.HasSuffix(r.URL.Path, wantPath) || r.URL.Query().Get("region") != "us-east-1" {
				t.Errorf("unexpected scheduled-search write endpoint: %s?%s", r.URL.Path, r.URL.RawQuery)
			}
			writes++
			if err := json.NewDecoder(r.Body).Decode(&stored); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored["id"] = json.RawMessage(`"search-1"`)
			fmt.Fprint(w, mustJSON(t, stored))
		case http.MethodGet:
			if !strings.HasSuffix(r.URL.Path, "/logs_settings/scheduled_search") || r.URL.Query().Get("region") != "us-east-1" {
				t.Errorf("unexpected scheduled-search read endpoint: %s?%s", r.URL.Path, r.URL.RawQuery)
			}
			fmt.Fprintf(w, "[%s]", mustJSON(t, stored))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	r := resourceScheduledSearchAlert()
	config := scheduledSearchResourceConfig("search", true)
	state := applyScheduledSearchConfig(t, r, nil, config, c)
	if got := scheduledSearchResultant(t, stored); got != resultant {
		t.Fatalf("create sent resultant_query %q, want %q", got, resultant)
	}
	importData := schema.TestResourceDataRaw(t, r.Schema, nil)
	importData.SetId(state.ID)
	imported, err := r.Importer.StateContext(context.Background(), importData, c)
	if err != nil {
		t.Fatal(err)
	}
	state, diags := r.RefreshWithoutUpgrade(context.Background(), imported[0].State(), c)
	if diags.HasError() {
		t.Fatal(diags)
	}

	config = scheduledSearchResourceConfig("renamed search", false)
	state = applyScheduledSearchConfig(t, r, state, config, c)
	if got := scheduledSearchResultant(t, stored); got != resultant {
		t.Fatalf("unrelated update sent resultant_query %q, want preserved pipeline %q", got, resultant)
	}

	const replacement = `[{"type":"filter","query":{"$eq":["service","worker"]}},{"type":"aggregate","aggregates":[{"function":{"$count":[]},"as":"result"}],"groupby":{}}]`
	config = scheduledSearchResourceConfig("renamed search", true)
	config["query"] = `[{"type":"filter","query":{"$eq":["service","worker"]}}]`
	config["resultant_query"] = replacement
	state = applyScheduledSearchConfig(t, r, state, config, c)
	if got := scheduledSearchResultant(t, stored); got != replacement {
		t.Fatalf("replacement update sent resultant_query %q, want %q", got, replacement)
	}
	if writes != 3 {
		t.Fatalf("expected create and two updates, got %d", writes)
	}

	delete(config, "resultant_query")
	diff, err := r.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), c)
	if err != nil {
		t.Fatal(err)
	}
	if diff != nil && !diff.Empty() {
		t.Fatalf("plan after update should be empty, got %#v", diff)
	}
}

func TestScheduledSearchAlertRejectsSourceChangeWithoutPipeline(t *testing.T) {
	calls := 0
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Errorf("invalid update must not write: %s %s", r.Method, r.URL.Path)
	})
	r := resourceScheduledSearchAlert()
	state := schema.TestResourceDataRaw(t, r.Schema, scheduledSearchResourceConfig("search", true))
	state.SetId("us-east-1:search-1:search")
	state.Set("resultant_query", `[{"type":"aggregate"}]`)
	config := scheduledSearchResourceConfig("search", false)
	config["query"] = `[{"type":"filter","query":{"$eq":["service","worker"]}}]`
	diff, err := r.Diff(context.Background(), state.State(), terraform.NewResourceConfigRaw(config), c)
	if err != nil {
		t.Fatal(err)
	}
	diff.RawConfig, err = schema.JSONMapToStateValue(config, r.CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}
	_, diags := r.Apply(context.Background(), state.State(), diff, c)
	if !diags.HasError() || !strings.Contains(diags[0].Summary+diags[0].Detail, "resultant_query") {
		t.Fatalf("expected actionable resultant_query error, got %v", diags)
	}
	if calls != 0 {
		t.Fatalf("expected no API writes, got %d", calls)
	}
}

func TestScheduledSearchAlertRejectsPostProcessorChangeWithoutPipeline(t *testing.T) {
	calls := 0
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Errorf("invalid update must not write: %s %s", r.Method, r.URL.Path)
	})
	r := resourceScheduledSearchAlert()
	state := schema.TestResourceDataRaw(t, r.Schema, scheduledSearchResourceConfig("search", true))
	state.SetId("us-east-1:search-1:search")
	config := scheduledSearchResourceConfig("search", false)
	config["post_processor"] = []interface{}{map[string]interface{}{"type": "aggregate", "aggregates": []interface{}{map[string]interface{}{"function": `{"$sum":["latency"]}`, "as": "total"}}, "groupby": "{}"}}
	diff, err := r.Diff(context.Background(), state.State(), terraform.NewResourceConfigRaw(config), c)
	if err != nil {
		t.Fatal(err)
	}
	diff.RawConfig, err = schema.JSONMapToStateValue(config, r.CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}
	_, diags := r.Apply(context.Background(), state.State(), diff, c)
	if !diags.HasError() || !strings.Contains(diags[0].Summary+diags[0].Detail, "resultant_query") {
		t.Fatalf("expected actionable resultant_query error, got %v", diags)
	}
	if calls != 0 {
		t.Fatalf("expected no API writes, got %d", calls)
	}
}

func TestScheduledSearchAlertCreateWithoutPipelineDoesNotWrite(t *testing.T) {
	calls := 0
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Errorf("aggregate create without resultant_query must not write: %s %s", r.Method, r.URL.Path)
	})
	r := resourceScheduledSearchAlert()
	config := scheduledSearchResourceConfig("search", false)
	diff, err := r.Diff(context.Background(), nil, terraform.NewResourceConfigRaw(config), c)
	if err != nil {
		t.Fatal(err)
	}
	diff.RawConfig, err = schema.JSONMapToStateValue(config, r.CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}
	_, diags := r.Apply(context.Background(), nil, diff, c)
	if !diags.HasError() || !strings.Contains(diags[0].Summary+diags[0].Detail, "resultant_query") {
		t.Fatalf("expected resultant_query error, got %v", diags)
	}
	if calls != 0 {
		t.Fatalf("expected no API writes, got %d", calls)
	}
}

func TestScheduledSearchAlertInvalidPipelineDoesNotWrite(t *testing.T) {
	for _, pipeline := range []string{"   ", "null", "{}", "[]", `[{"type":"aggregate"}, null]`, `["filter"]`} {
		t.Run(pipeline, func(t *testing.T) {
			calls := 0
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				t.Errorf("invalid resultant_query must not write: %s %s", r.Method, r.URL.Path)
			})
			d := schema.TestResourceDataRaw(t, resourceScheduledSearchAlert().Schema, scheduledSearchResourceConfig("search", false))
			d.Set("resultant_query", pipeline)
			diags := resourceScheduledSearchAlertCreate(context.Background(), d, c)
			if !diags.HasError() || !strings.Contains(diags[0].Summary+diags[0].Detail, "resultant_query") {
				t.Fatalf("expected resultant_query validation error for %q, got %v", pipeline, diags)
			}
			if calls != 0 {
				t.Fatalf("expected no API writes, got %d", calls)
			}
		})
	}
}

func TestScheduledSearchAlertPlanDistinguishesOmittedAndUnknownPipeline(t *testing.T) {
	r := resourceScheduledSearchAlert()
	server := schema.NewGRPCProviderServer(&schema.Provider{ResourcesMap: map[string]*schema.Resource{"last9_scheduled_search_alert": r}})
	schemaBlock := r.CoreConfigSchema()
	typeSchema := schemaBlock.ImpliedType()
	priorState, err := msgpack.Marshal(cty.NullVal(typeSchema), typeSchema)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		pipeline *cty.Value
		wantErr  bool
	}{
		{name: "omitted", wantErr: true},
		{name: "explicit null", pipeline: ctyValuePtr(cty.NullVal(cty.String)), wantErr: true},
		{name: "unknown", pipeline: ctyValuePtr(cty.UnknownVal(cty.String))},
		{name: "valid", pipeline: ctyValuePtr(cty.StringVal(`[{"type":"aggregate"}]`))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := scheduledSearchResourceConfig("search", false)
			configValue, err := schema.JSONMapToStateValue(config, schemaBlock)
			if err != nil {
				t.Fatal(err)
			}
			if tc.pipeline != nil {
				values := configValue.AsValueMap()
				values["resultant_query"] = *tc.pipeline
				configValue = cty.ObjectVal(values)
			}
			configBytes, err := msgpack.Marshal(configValue, typeSchema)
			if err != nil {
				t.Fatal(err)
			}
			proposedState, err := msgpack.Marshal(configValue, typeSchema)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := server.PlanResourceChange(context.Background(), &tfprotov5.PlanResourceChangeRequest{
				TypeName:         "last9_scheduled_search_alert",
				PriorState:       &tfprotov5.DynamicValue{MsgPack: priorState},
				ProposedNewState: &tfprotov5.DynamicValue{MsgPack: proposedState},
				Config:           &tfprotov5.DynamicValue{MsgPack: configBytes},
			})
			if err != nil {
				t.Fatal(err)
			}
			hasError := false
			for _, diagnostic := range resp.Diagnostics {
				if diagnostic.Severity == tfprotov5.DiagnosticSeverityError {
					hasError = true
					if !strings.Contains(diagnostic.Summary+diagnostic.Detail, "resultant_query") {
						t.Errorf("unexpected planning diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
					}
				}
			}
			if hasError != tc.wantErr {
				t.Fatalf("has planning error=%v, want %v; diagnostics: %+v", hasError, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func ctyValuePtr(value cty.Value) *cty.Value { return &value }

func scheduledSearchResourceConfig(name string, includePipeline bool) map[string]interface{} {
	config := map[string]interface{}{
		"region":             "us-east-1",
		"name":               name,
		"query_type":         "logjson-aggregate",
		"physical_index":     "logs",
		"telemetry":          "logs",
		"query":              `[{"type":"filter","query":{"$eq":["service","api"]}}]`,
		"post_processor":     []interface{}{map[string]interface{}{"type": "aggregate", "aggregates": []interface{}{map[string]interface{}{"function": `{"$count":[]}`, "as": "count"}}, "groupby": "{}"}},
		"search_frequency":   300,
		"threshold":          []interface{}{map[string]interface{}{"operator": ">", "value": 10.0}},
		"alert_destinations": []interface{}{},
	}
	if includePipeline {
		config["resultant_query"] = `[{"type":"filter","query":{"$eq":["service","api"]}},{"type":"aggregate","aggregates":[{"function":{"$count":[]},"as":"result"}],"groupby":{}}]`
	}
	return config
}

func applyScheduledSearchConfig(t *testing.T, r *schema.Resource, state *terraform.InstanceState, config map[string]interface{}, c interface{}) *terraform.InstanceState {
	t.Helper()
	var prior *terraform.InstanceState
	if state != nil {
		prior = state
	}
	diff, err := r.Diff(context.Background(), prior, terraform.NewResourceConfigRaw(config), c)
	if err != nil {
		t.Fatal(err)
	}
	diff.RawConfig, err = schema.JSONMapToStateValue(config, r.CoreConfigSchema())
	if err != nil {
		t.Fatal(err)
	}
	state, diags := r.Apply(context.Background(), prior, diff, c)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func scheduledSearchResultant(t *testing.T, stored map[string]json.RawMessage) string {
	t.Helper()
	var response struct {
		Properties struct {
			ResultantQuery string `json:"resultant_query"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(mustJSON(t, stored)), &response); err != nil {
		t.Fatal(err)
	}
	return response.Properties.ResultantQuery
}

func mustJSON(t *testing.T, value interface{}) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
