package provider

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/last9/terraform-provider-last9/internal/client"
)

// fakeNotificationServer serves ListNotificationDestinations/
// GetEntityNotificationBindings from a fixed catalog and records every
// attach/detach call it receives, so a test can assert reconcile logic made
// exactly the calls it should — no more (don't touch what's already
// correct) and no less (don't leave a wanted channel unbound or a removed
// one still bound).
type fakeNotificationServer struct {
	mu       sync.Mutex
	catalog  []client.NotificationDestination // full org channel list
	live     map[string]client.NotificationDestination
	attached []string // channel names attached during the test
	detached []int    // binding row IDs detached during the test
}

func newFakeNotificationServer(catalog []client.NotificationDestination, live map[string]client.NotificationDestination) *fakeNotificationServer {
	return &fakeNotificationServer{catalog: catalog, live: live}
}

func (f *fakeNotificationServer) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet:
			// Mirrors the real API's confirmed production behavior (verified
			// directly against a live tenant): GET /notification_settings
			// with NO entity_id query param returns only the org's
			// master/global catalog rows and NEVER a per-entity bound row,
			// even when one exists. Only GET /notification_settings?entity_id=<id>
			// includes that entity's bound rows (mixed in with the
			// masters). ListNotificationDestinations (name/ID resolution)
			// hits the unfiltered path; GetEntityNotificationBindings hits
			// the entity_id-filtered one. A fake server that ignored this
			// distinction previously let GetEntityNotificationBindings
			// silently revert to the unfiltered endpoint and still pass
			// every test, while finding zero bindings against any real
			// tenant.
			entityID := r.URL.Query().Get("entity_id")
			result := make([]client.NotificationDestination, 0, len(f.catalog))
			result = append(result, f.catalog...)
			if entityID != "" {
				for _, b := range f.live {
					if b.ServiceFqid == entityID {
						result = append(result, b)
					}
				}
			}
			_ = json.NewEncoder(w).Encode(result)

		case r.Method == http.MethodPost:
			var body client.AttachNotificationSettingsRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			// path: /notification_settings/{channelID}/attach
			var channelID int
			for _, c := range f.catalog {
				if r.URL.Path == "/api/v4/organizations/test-org/notification_settings/"+strconv.Itoa(c.ID)+"/attach" {
					channelID = c.ID
					name := c.Name
					f.attached = append(f.attached, name)
					f.live[name] = client.NotificationDestination{
						ID:          channelID + 100000, // distinct binding-row id from the master channel id
						Name:        name,
						ServiceFqid: body.EntityID,
						Severity:    body.Severity,
					}
					break
				}
			}
			_ = json.NewEncoder(w).Encode(f.live)

		case r.Method == http.MethodDelete:
			for name, b := range f.live {
				if r.URL.Path == "/api/v4/organizations/test-org/notification_settings/"+strconv.Itoa(b.ID)+"/attach" {
					f.detached = append(f.detached, b.ID)
					delete(f.live, name)
					break
				}
			}
			w.WriteHeader(http.StatusOK)
		}
	}
}
