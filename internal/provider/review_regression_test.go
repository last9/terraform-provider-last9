package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/last9/terraform-provider-last9/internal/client"
)

func reviewClient(t *testing.T, h http.HandlerFunc) *client.Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	c, err := client.NewClient(&client.Config{
		Org:         "synthetic",
		APIToken:    "synthetic-token",
		DeleteToken: "synthetic-delete",
		BaseURL:     s.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestReviewUserCreateHonorsInactive(t *testing.T) {
	for _, active := range []bool{true, false} {
		t.Run(fmt.Sprint(active), func(t *testing.T) {
			patches := 0
			deleted := false
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "POST" && r.URL.Path == "/api/v4/organizations/synthetic/users/invite":
					fmt.Fprint(w, `{}`)
				case r.Method == "PATCH" && r.URL.Path == "/api/v4/organizations/synthetic/users/u1":
					var b struct {
						Active bool `json:"active"`
					}
					if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
						t.Error(e)
					}
					deleted = !b.Active
					patches++
					fmt.Fprint(w, `{}`)
				case r.Method == "GET" && r.URL.Path == "/api/v4/organizations/synthetic/users":
					at := "null"
					if deleted {
						at = "1"
					}
					fmt.Fprintf(w, `[{"id":"u1","email":"review@example.test","role":"viewer","status":"invited","deleted_at":%s}]`, at)
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			d := schema.TestResourceDataRaw(t, resourceUser().Schema, map[string]interface{}{
				"email":  "review@example.test",
				"role":   "viewer",
				"active": active,
			})
			if ds := resourceUserCreate(context.Background(), d, c); ds.HasError() {
				t.Fatal(ds)
			}
			if !active && patches != 1 {
				t.Errorf("inactive invite must be deactivated: PATCH calls=%d want=1", patches)
			}
			if got := d.Get("active").(bool); got != active {
				t.Errorf("active=%v want=%v", got, active)
			}
		})
	}
}

func TestReviewPhysicalIndexRejectedDeletePreservesState(t *testing.T) {
	for _, code := range []int{204, 400} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" || r.URL.Query().Get("cluster_id") != "c1" {
					t.Errorf("unexpected %s %s", r.Method, r.URL)
				}
				w.WriteHeader(code)
				if code == 400 {
					fmt.Fprint(w, `{"error":"physical index is not created yet, cannot soft-delete it"}`)
				}
			})
			d := schema.TestResourceDataRaw(t, resourcePhysicalIndex().Schema, map[string]interface{}{})
			d.SetId("us-east-1:c1:i1")
			// Avoid long retry backoff in unit test for 400 path.
			ctx := context.Background()
			if code == 400 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel() // cancel immediately so retries exit fast via ctx.Done
			}
			ds := resourcePhysicalIndexDelete(ctx, d, c)
			if code == 400 && (!ds.HasError() || d.Id() == "") {
				t.Fatalf("rejected delete must retain retryable state: diagnostics=%v id=%q", ds, d.Id())
			}
			if code == 204 && (ds.HasError() || d.Id() != "") {
				t.Fatalf("successful delete: diagnostics=%v id=%q", ds, d.Id())
			}
		})
	}
}

func TestReviewSyntheticMaskedHeadersPreserveConfiguredValue(t *testing.T) {
	const configured = `{"url":"https://example.test","headers":{"Authorization":"Bearer synthetic-fixture"}}`
	for _, masked := range []bool{false, true} {
		t.Run(fmt.Sprint(masked), func(t *testing.T) {
			remote := configured
			if masked {
				remote = `{"url":"https://example.test","headers":{"Authorization":"********"}}`
			}
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("unexpected method %s", r.Method)
				}
				fmt.Fprintf(w, `{"check":{"id":"c1","name":"test","type":"http","status":"active","schedule":"every 5m","timeout":30,"frequency":60,"locations":["us-east-1"],"config":%s}}`, remote)
			})
			d := schema.TestResourceDataRaw(t, resourceSyntheticCheck().Schema, map[string]interface{}{
				"name":      "test",
				"type":      "http",
				"schedule":  "every 5m",
				"timeout":   30,
				"frequency": 60,
				"locations": []interface{}{"us-east-1"},
				"config":    configured,
			})
			d.SetId("c1")
			if ds := resourceSyntheticCheckRead(context.Background(), d, c); ds.HasError() {
				t.Fatal(ds)
			}
			got := d.Get("config").(string)
			if masked {
				var m map[string]interface{}
				if err := json.Unmarshal([]byte(got), &m); err != nil {
					t.Fatal(err)
				}
				headers := m["headers"].(map[string]interface{})
				if headers["Authorization"] != "Bearer synthetic-fixture" {
					t.Fatalf("masked header not preserved: %s", got)
				}
			} else if !suppressEquivalentJSON("", configured, got, d) {
				t.Fatalf("unmasked config drift: %s", got)
			}
		})
	}
}

func TestReviewSnoozePreservesConfiguredUntilAfterRemoteClear(t *testing.T) {
	for _, remote := range []int{1893456000, 0} {
		t.Run(fmt.Sprint(remote), func(t *testing.T) {
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `{"alert_snoozed_until":%d}`, remote)
			})
			resource := resourceAlertSnooze()
			config := map[string]interface{}{"entity_id": "entity-a", "until": 1893456000}
			d := schema.TestResourceDataRaw(t, resource.Schema, config)
			d.SetId("entity-a")
			if ds := resource.ReadContext(context.Background(), d, c); ds.HasError() {
				t.Fatal(ds)
			}
			diff, e := resource.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), c)
			if e != nil {
				t.Fatal(e)
			}
			change := diff != nil && diff.Attributes["until"] != nil
			if change {
				t.Fatalf("refresh changed configured until after API returned %d", remote)
			}
		})
	}
}

func TestReviewDatasourceDefaultWireContract(t *testing.T) {
	for _, byID := range []bool{true, false} {
		t.Run(fmt.Sprint(byID), func(t *testing.T) {
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/api/v4/organizations/synthetic/datasources" {
					t.Errorf("unexpected %s %s", r.Method, r.URL)
				}
				fmt.Fprint(w, `[{"id":"secondary","name":"secondary","type":"prometheus","region":"us-east-1","is_default":false},{"id":"primary","name":"primary","type":"prometheus","region":"us-east-1","is_default":true}]`)
			})
			input := map[string]interface{}{}
			if byID {
				input["id"] = "primary"
			}
			d := schema.TestResourceDataRaw(t, dataSourceDatasource().Schema, input)
			if ds := dataSourceDatasourceRead(context.Background(), d, c); ds.HasError() {
				t.Fatal(ds)
			}
			if d.Id() != "primary" {
				t.Fatalf("default lookup selected %q; want primary", d.Id())
			}
			if !byID && d.Get("default") != true {
				t.Fatalf("default flag=%v want true", d.Get("default"))
			}
		})
	}
}

func TestReviewColdStorageBucketValidatesAuthentication(t *testing.T) {
	base := map[string]interface{}{
		"region": "us-east-1", "name": "bucket", "aws_region": "us-east-1", "aws_bucket": "bucket",
	}
	for name, tc := range map[string]struct {
		values map[string]interface{}
		valid  bool
	}{
		"credentials require both keys": {map[string]interface{}{"auth_type": "credentials", "aws_access_key": "key"}, false},
		"credentials reject role":       {map[string]interface{}{"auth_type": "credentials", "aws_access_key": "key", "aws_secret_key": "secret", "aws_role": "role"}, false},
		"role requires role":            {map[string]interface{}{"auth_type": "role"}, false},
		"role rejects credentials":      {map[string]interface{}{"auth_type": "role", "aws_role": "role", "aws_access_key": "key", "aws_secret_key": "secret"}, false},
		"valid credentials":             {map[string]interface{}{"auth_type": "credentials", "aws_access_key": "key", "aws_secret_key": "secret"}, true},
		"valid role":                    {map[string]interface{}{"auth_type": "role", "aws_role": "role"}, true},
	} {
		t.Run(name, func(t *testing.T) {
			config := make(map[string]interface{}, len(base)+len(tc.values))
			for k, v := range base {
				config[k] = v
			}
			for k, v := range tc.values {
				config[k] = v
			}
			_, err := resourceColdStorageBucket().Diff(context.Background(), nil, terraform.NewResourceConfigRaw(config), nil)
			if (err == nil) != tc.valid {
				t.Fatalf("validation error = %v, valid = %v", err, tc.valid)
			}
		})
	}
}

func TestReviewColdStorageBackupValidatesTargets(t *testing.T) {
	base := map[string]interface{}{"region": "us-east-1", "name": "backup", "bucket_name": "bucket"}
	for name, tc := range map[string]map[string]interface{}{
		"service requires targets": {"granularity": "service"},
		"index rejects targets":    {"granularity": "index", "targets": []interface{}{"index-a"}},
		"service accepts targets":  {"granularity": "service", "targets": []interface{}{"service-a"}},
		"index accepts no targets": {"granularity": "index"},
	} {
		t.Run(name, func(t *testing.T) {
			config := make(map[string]interface{}, len(base)+len(tc))
			for k, v := range base {
				config[k] = v
			}
			for k, v := range tc {
				config[k] = v
			}
			_, err := resourceColdStorageBackup().Diff(context.Background(), nil, terraform.NewResourceConfigRaw(config), nil)
			valid := name == "service accepts targets" || name == "index accepts no targets"
			if (err == nil) != valid {
				t.Fatalf("validation error = %v, valid = %v", err, valid)
			}
		})
	}
}

func TestReviewUnknownColdStorageInputsCanPlan(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66"
	for _, tc := range []struct {
		name     string
		resource *schema.Resource
		config   map[string]interface{}
	}{
		{"role ARN from new IAM role", resourceColdStorageBucket(), map[string]interface{}{
			"region": "us-east-1", "name": "archive", "aws_region": "us-east-1", "aws_bucket": "synthetic-archive", "auth_type": "role", "aws_role": unknown,
		}},
		{"credentials from new access key", resourceColdStorageBucket(), map[string]interface{}{
			"region": "us-east-1", "name": "archive", "aws_region": "us-east-1", "aws_bucket": "synthetic-archive", "auth_type": "credentials", "aws_access_key": unknown, "aws_secret_key": unknown,
		}},
		{"service targets from computed list", resourceColdStorageBackup(), map[string]interface{}{
			"region": "us-east-1", "name": "backup", "bucket_name": "archive", "granularity": "service", "targets": unknown,
		}},
		{"known role", resourceColdStorageBucket(), map[string]interface{}{
			"region": "us-east-1", "name": "archive", "aws_region": "us-east-1", "aws_bucket": "synthetic-archive", "auth_type": "role", "aws_role": "arn:aws:iam::123456789012:role/synthetic",
		}},
		{"known targets", resourceColdStorageBackup(), map[string]interface{}{
			"region": "us-east-1", "name": "backup", "bucket_name": "archive", "granularity": "service", "targets": []interface{}{"api"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.resource.Diff(context.Background(), nil, terraform.NewResourceConfigRaw(tc.config), nil); err != nil {
				t.Fatalf("valid configuration must plan with apply-time inputs: %v", err)
			}
		})
	}
}

func TestReviewDataSourcesRejectAmbiguousSelectors(t *testing.T) {
	for name, tc := range map[string]map[string]interface{}{
		"cluster":    {"region": "us-east-1", "id": "cluster", "name": "cluster"},
		"datasource": {"id": "datasource", "name": "datasource"},
		"user":       {"id": "user", "email": "user@example.test"},
	} {
		t.Run(name, func(t *testing.T) {
			if diags := New().ValidateDataSource("last9_"+strings.ReplaceAll(name, " ", "_"), terraform.NewResourceConfigRaw(tc)); !diags.HasError() {
				t.Fatal("ambiguous selectors were accepted")
			}
		})
	}
}

func TestReviewChangeboardEmptyFieldsSerializeForRemoval(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceChangeboard().Schema, map[string]interface{}{
		"name": "board", "owner_id": "owner", "owner_type": "team",
	})
	var got map[string]interface{}
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Error(err)
			}
			fmt.Fprint(w, `{}`)
		case "GET":
			fmt.Fprint(w, `{"id":"board","name":"board","owner_id":"owner","owner_type":"team","filters":[],"groups":[],"relationships":[],"properties":{"granularity":""}}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	d.SetId("board")
	if ds := resourceChangeboardUpdate(context.Background(), d, c); ds.HasError() {
		t.Fatal(ds)
	}
	if relationships, ok := got["relationships"].([]interface{}); !ok || len(relationships) != 0 {
		t.Fatalf("relationships must serialize empty to clear remote state: %#v", got)
	}
	properties, ok := got["properties"].(map[string]interface{})
	if !ok || properties["granularity"] != "" {
		t.Fatalf("granularity must serialize empty to clear remote state: %#v", got)
	}
}

func TestReviewChangeboardRemovingGranularityPlansEmptyUpdate(t *testing.T) {
	resource := resourceChangeboard()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"name": "board", "owner_id": "owner", "owner_type": "team", "granularity": "daily",
	})
	d.SetId("board")
	diff, err := resource.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(map[string]interface{}{
		"name": "board", "owner_id": "owner", "owner_type": "team",
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil || diff.Attributes["granularity"] == nil {
		t.Fatal("removing granularity must plan an API update")
	}
}

func TestReviewPhysicalIndexRemovingRetentionSerializesNull(t *testing.T) {
	for _, tc := range []struct {
		name      string
		config    map[string]interface{}
		wantValue interface{}
		rawValue  cty.Value
	}{
		{
			name: "removed",
			config: map[string]interface{}{
				"region": "us-east-1", "name": "logs", "telemetry": "logs", "filters": []interface{}{map[string]interface{}{"key": "service.name", "operator": "equals", "value": "api"}},
			},
			wantValue: nil,
			rawValue:  cty.NullVal(cty.Number),
		},
		{
			name: "configured zero",
			config: map[string]interface{}{
				"region": "us-east-1", "name": "logs", "telemetry": "logs", "filters": []interface{}{map[string]interface{}{"key": "service.name", "operator": "equals", "value": "api"}}, "retention_period": 0,
			},
			wantValue: float64(0),
			rawValue:  cty.NumberIntVal(0),
		},
		{
			name: "configured positive",
			config: map[string]interface{}{
				"region": "us-east-1", "name": "logs", "telemetry": "logs", "filters": []interface{}{map[string]interface{}{"key": "service.name", "operator": "equals", "value": "api"}}, "retention_period": 14,
			},
			wantValue: float64(14),
			rawValue:  cty.NumberIntVal(14),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var put map[string]interface{}
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPut:
					if err := json.NewDecoder(r.Body).Decode(&put); err != nil {
						t.Error(err)
					}
					fmt.Fprint(w, `{"id":"logs"}`)
				case http.MethodGet:
					fmt.Fprint(w, `{"id":"logs","name":"logs","properties":{"telemetry":"logs","filters":[{"key":"service.name","operator":"equals","value":"api"}],"retain":false},"status":"active"}`)
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusInternalServerError)
				}
			})
			resource := resourcePhysicalIndex()
			stateData := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
				"region": "us-east-1", "name": "logs", "telemetry": "logs", "filters": []interface{}{map[string]interface{}{"key": "service.name", "operator": "equals", "value": "api"}}, "retention_period": 7,
			})
			stateData.SetId("us-east-1:cluster:logs")
			diff, err := resource.Diff(context.Background(), stateData.State(), terraform.NewResourceConfigRaw(tc.config), c)
			if err != nil {
				t.Fatal(err)
			}
			if diff == nil || diff.Empty() {
				t.Fatal("retention change must produce an update")
			}
			diff.RawConfig = cty.ObjectVal(map[string]cty.Value{"retention_period": tc.rawValue})
			if _, ds := resource.Apply(context.Background(), stateData.State(), diff, c); ds.HasError() {
				t.Fatal(ds)
			}
			properties, ok := put["properties"].(map[string]interface{})
			if !ok {
				t.Fatalf("PUT is missing properties: %#v", put)
			}
			if got, ok := properties["retention_period"]; !ok || got != tc.wantValue {
				t.Fatalf("retention_period must be present as %#v: %#v", tc.wantValue, put)
			}
		})
	}
}

func TestReviewColdStorageBucketDefaultFalseIsSerialized(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceColdStorageBucket().Schema, map[string]interface{}{
		"region": "us-east-1", "name": "bucket", "aws_region": "us-east-1", "aws_bucket": "bucket", "auth_type": "role", "aws_role": "role", "default": false,
	})
	d.SetId("us-east-1:bucket-1")
	var put map[string]interface{}
	c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			if err := json.NewDecoder(r.Body).Decode(&put); err != nil {
				t.Error(err)
			}
			fmt.Fprint(w, `{}`)
		case "GET":
			fmt.Fprint(w, `{"id":"bucket-1","name":"bucket","properties":{"default":false,"aws_region":"us-east-1","aws_bucket":"bucket","auth_type":"role","aws_role":"role"},"status":"active"}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	if ds := resourceColdStorageBucketUpdate(context.Background(), d, c); ds.HasError() {
		t.Fatal(ds)
	}
	if got, ok := put["properties"].(map[string]interface{})["default"].(bool); !ok || got {
		t.Fatalf("default=false must be sent to the bucket PUT endpoint: %#v", put)
	}
}

func TestReviewSyntheticCreateHonorsPaused(t *testing.T) {
	for _, status := range []string{"active", "paused"} {
		t.Run(status, func(t *testing.T) {
			createdStatus := "active"
			updated := false
			c := reviewClient(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/synthetic/checks"):
					var body map[string]interface{}
					_ = json.NewDecoder(r.Body).Decode(&body)
					if s, ok := body["status"].(string); ok && s != "" {
						createdStatus = s
					} else {
						createdStatus = "active"
					}
					fmt.Fprintf(w, `{"check":{"id":"c1","name":"test","type":"http","status":%q,"schedule":"every 5m","timeout":30,"frequency":60,"locations":["us-east-1"],"config":{"url":"https://example.test"}}}`+"\n", createdStatus)
				case r.Method == "PUT" && strings.Contains(r.URL.Path, "/synthetic/checks/c1"):
					var body map[string]interface{}
					_ = json.NewDecoder(r.Body).Decode(&body)
					if s, ok := body["status"].(string); ok {
						createdStatus = s
						updated = true
					}
					fmt.Fprintf(w, `{"check":{"id":"c1","name":"test","type":"http","status":%q,"schedule":"every 5m","timeout":30,"frequency":60,"locations":["us-east-1"],"config":{"url":"https://example.test"}}}`+"\n", createdStatus)
				case r.Method == "GET" && strings.Contains(r.URL.Path, "/synthetic/checks/c1"):
					fmt.Fprintf(w, `{"check":{"id":"c1","name":"test","type":"http","status":%q,"schedule":"every 5m","timeout":30,"frequency":60,"locations":["us-east-1"],"config":{"url":"https://example.test"}}}`+"\n", createdStatus)
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			d := schema.TestResourceDataRaw(t, resourceSyntheticCheck().Schema, map[string]interface{}{
				"name":      "test",
				"type":      "http",
				"schedule":  "every 5m",
				"timeout":   30,
				"frequency": 60,
				"locations": []interface{}{"us-east-1"},
				"config":    `{"url":"https://example.test"}`,
				"status":    status,
			})
			if ds := resourceSyntheticCheckCreate(context.Background(), d, c); ds.HasError() {
				t.Fatal(ds)
			}
			if got := d.Get("status").(string); got != status {
				t.Fatalf("status=%q want=%q (updated=%v)", got, status, updated)
			}
		})
	}
}

func TestReviewNewExamplesParseAsTerraform(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../.."))
	for _, name := range []string{"cold-storage", "streaming-aggregations", "synthetics"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "examples", name, "main.tf")
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, ds := hclparse.NewParser().ParseHCL(src, path); ds.HasErrors() {
				t.Fatalf("example must be valid HCL: %s", ds.Error())
			}
		})
	}
}
