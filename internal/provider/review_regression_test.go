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

func TestReviewSnoozePlansRepairAfterRemoteClear(t *testing.T) {
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
			if remote == 0 && !change {
				t.Fatal("remote snooze cleared, but plan contains no until repair")
			}
			if remote == 1893456000 && change {
				t.Fatal("unchanged control must not repair until")
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
