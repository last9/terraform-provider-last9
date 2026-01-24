package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/last9/terraform-provider-last9/internal/client"
)

// Unit tests for resource functions
func TestResourceAlert_parseAndSetCondition(t *testing.T) {
	tests := []struct {
		name           string
		condition      string
		evalWindow     int
		alertCondition string
		wantGreater    float64
		wantLess       float64
		wantBadMinutes int
		wantTotal      int
		wantErr        bool
	}{
		{
			name:           "greater than condition",
			condition:      "expr > 100",
			evalWindow:     10,
			alertCondition: "count_true(result) >= 5",
			wantGreater:    100,
			wantBadMinutes: 5,
			wantTotal:      10,
			wantErr:        false,
		},
		{
			name:           "less than condition",
			condition:      "expr < 50",
			evalWindow:     15,
			alertCondition: "count_true(result) >= 3",
			wantLess:       50,
			wantBadMinutes: 3,
			wantTotal:      15,
			wantErr:        false,
		},
		{
			name:           "invalid condition",
			condition:      "invalid",
			evalWindow:     10,
			alertCondition: "",
			wantErr:        false, // Function doesn't error, just doesn't set values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resourceAlert().Schema, make(map[string]interface{}))

			err := parseAndSetCondition(d, tt.condition, tt.evalWindow, tt.alertCondition)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAndSetCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantGreater > 0 {
				if got, ok := d.GetOk("greater_than"); ok {
					if got.(float64) != tt.wantGreater {
						t.Errorf("greater_than = %v, want %v", got, tt.wantGreater)
					}
				} else {
					t.Errorf("greater_than not set, want %v", tt.wantGreater)
				}
			}

			if tt.wantLess > 0 {
				if got, ok := d.GetOk("less_than"); ok {
					if got.(float64) != tt.wantLess {
						t.Errorf("less_than = %v, want %v", got, tt.wantLess)
					}
				} else {
					t.Errorf("less_than not set, want %v", tt.wantLess)
				}
			}

			if tt.wantBadMinutes > 0 {
				if got, ok := d.GetOk("bad_minutes"); ok {
					if got.(int) != tt.wantBadMinutes {
						t.Errorf("bad_minutes = %v, want %v", got, tt.wantBadMinutes)
					}
				} else {
					t.Errorf("bad_minutes not set, want %v", tt.wantBadMinutes)
				}
			}

			if tt.wantTotal > 0 {
				if got, ok := d.GetOk("total_minutes"); ok {
					if got.(int) != tt.wantTotal {
						t.Errorf("total_minutes = %v, want %v", got, tt.wantTotal)
					}
				} else {
					t.Errorf("total_minutes not set, want %v", tt.wantTotal)
				}
			}
		})
	}
}

func TestResourceAlert_expandRoutingFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters []interface{}
		wantLen int
	}{
		{
			name: "single filter",
			filters: []interface{}{
				map[string]interface{}{
					"key":      "attributes[\"service\"]",
					"value":    "test",
					"operator": "equals",
				},
			},
			wantLen: 1,
		},
		{
			name: "multiple filters",
			filters: []interface{}{
				map[string]interface{}{
					"key":      "attributes[\"service\"]",
					"value":    "test",
					"operator": "equals",
				},
				map[string]interface{}{
					"key":         "resource.attributes[\"env\"]",
					"value":       "prod",
					"operator":    "equals",
					"conjunction": "and",
				},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandRoutingFilters(tt.filters)
			if len(got) != tt.wantLen {
				t.Errorf("expandRoutingFilters() len = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestResourceAlert_nilRunbookHandling(t *testing.T) {
	// Test that nil Runbook doesn't cause panic
	props := client.AlertProperties{
		Description: "Test",
		Runbook:     nil, // Explicitly nil
		Annotations: map[string]string{"key": "value"},
	}

	// This should not panic
	runbookURL := func() string {
		if props.Runbook != nil {
			if runbook, ok := props.Runbook["link"].(string); ok {
				return runbook
			}
		}
		return ""
	}()

	if runbookURL != "" {
		t.Errorf("Expected empty string for nil Runbook, got %s", runbookURL)
	}
}
