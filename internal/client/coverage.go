package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// --- Synthetics ---

type SyntheticCheck struct {
	ID             string            `json:"id,omitempty"`
	OrganizationID string            `json:"organization_id,omitempty"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Type           string            `json:"type"`
	Status         string            `json:"status,omitempty"`
	Schedule       string            `json:"schedule"`
	Config         json.RawMessage   `json:"config"`
	Timeout        int               `json:"timeout"`
	Frequency      int               `json:"frequency"`
	Locations      []string          `json:"locations"`
	Tags           map[string]string `json:"tags,omitempty"`
	CreatedBy      string            `json:"created_by,omitempty"`
	CreatedAt      string            `json:"created_at,omitempty"`
	UpdatedAt      string            `json:"updated_at,omitempty"`
}

type CreateSyntheticCheckRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type"`
	Schedule    string            `json:"schedule"`
	Config      json.RawMessage   `json:"config"`
	Timeout     int               `json:"timeout"`
	Frequency   int               `json:"frequency"`
	Locations   []string          `json:"locations"`
	Tags        map[string]string `json:"tags,omitempty"`
}

type UpdateSyntheticCheckRequest struct {
	Name        *string            `json:"name,omitempty"`
	Description *string            `json:"description,omitempty"`
	Status      *string            `json:"status,omitempty"`
	Schedule    *string            `json:"schedule,omitempty"`
	Config      *json.RawMessage   `json:"config,omitempty"`
	Timeout     *int               `json:"timeout,omitempty"`
	Frequency   *int               `json:"frequency,omitempty"`
	Locations   *[]string          `json:"locations,omitempty"`
	Tags        *map[string]string `json:"tags,omitempty"`
}

func (c *Client) CreateSyntheticCheck(req *CreateSyntheticCheckRequest) (*SyntheticCheck, error) {
	var wrap struct {
		Check SyntheticCheck `json:"check"`
	}
	err := c.Post("/synthetic/checks", req, &wrap)
	if err != nil {
		return nil, err
	}
	if wrap.Check.ID == "" {
		return nil, fmt.Errorf("create synthetic check: empty id in response")
	}
	return &wrap.Check, nil
}

func (c *Client) GetSyntheticCheck(id string) (*SyntheticCheck, error) {
	var wrap struct {
		Check SyntheticCheck `json:"check"`
	}
	err := c.Get(fmt.Sprintf("/synthetic/checks/%s", id), &wrap)
	if err != nil {
		return nil, err
	}
	if wrap.Check.ID == "" {
		return nil, fmt.Errorf("synthetic check %s not found", id)
	}
	return &wrap.Check, nil
}

func (c *Client) UpdateSyntheticCheck(id string, req *UpdateSyntheticCheckRequest) (*SyntheticCheck, error) {
	var wrap struct {
		Check SyntheticCheck `json:"check"`
	}
	err := c.Put(fmt.Sprintf("/synthetic/checks/%s", id), req, &wrap)
	if err != nil {
		return nil, err
	}
	if wrap.Check.ID != "" {
		return &wrap.Check, nil
	}
	return c.GetSyntheticCheck(id)
}

func (c *Client) DeleteSyntheticCheck(id string) error {
	return c.Delete(fmt.Sprintf("/synthetic/checks/%s", id))
}

// --- Relationships (client helpers retained; no TF resource) ---

type CreateRelationshipRequest struct {
	Destination string `json:"destination"`
	Type        string `json:"type"`
}

type CreateRelationshipResponse struct {
	ID string `json:"id"`
}

type UpdateRelationshipRequest struct {
	Type string `json:"type"`
}

type Relationship struct {
	ID           string `json:"id"`
	EntityName   string `json:"entity_name"`
	EntityType   string `json:"entity_type"`
	Relationship string `json:"relationship"`
	EntityID     string `json:"entity_id"`
	CreatedAt    int64  `json:"created_at"`
}

type RelationshipsResponse struct {
	Inbound  []Relationship `json:"inbound"`
	Outbound []Relationship `json:"outbound"`
}

func (c *Client) CreateRelationship(entityID string, req *CreateRelationshipRequest) (*CreateRelationshipResponse, error) {
	var result CreateRelationshipResponse
	err := c.Post(fmt.Sprintf("/entities/%s/relationships", entityID), req, &result)
	return &result, err
}

func (c *Client) ListRelationships(entityID string) (*RelationshipsResponse, error) {
	var result RelationshipsResponse
	err := c.Get(fmt.Sprintf("/entities/%s/relationships", entityID), &result)
	return &result, err
}

func (c *Client) UpdateRelationship(entityID, relationshipID string, req *UpdateRelationshipRequest) error {
	return c.Put(fmt.Sprintf("/entities/%s/relationships/%s", entityID, relationshipID), req, nil)
}

func (c *Client) DeleteRelationship(entityID, relationshipID string) error {
	return c.Delete(fmt.Sprintf("/entities/%s/relationships/%s", entityID, relationshipID))
}

// --- Changeboards ---

type ChangeBoardFilter struct {
	FilterType  string  `json:"filter_type"`
	Key         string  `json:"key"`
	Value       string  `json:"value,omitempty"`
	Operator    string  `json:"operator"`
	Conjunction *string `json:"conjunction,omitempty"`
}

type ChangeBoardGroup struct {
	Name  string `json:"name"`
	Order string `json:"order"`
}

type ChangeBoardRelationshipNode struct {
	ID       string                        `json:"id"`
	Children []ChangeBoardRelationshipNode `json:"children,omitempty"`
}

type ChangeBoardProperties struct {
	Granularity string `json:"granularity,omitempty"`
}

type ChangeBoardRequest struct {
	Name          string                        `json:"name"`
	Description   string                        `json:"description,omitempty"`
	Filters       []ChangeBoardFilter           `json:"filters"`
	Groups        []ChangeBoardGroup            `json:"groups"`
	OwnerID       string                        `json:"owner_id"`
	OwnerType     string                        `json:"owner_type"`
	Relationships []ChangeBoardRelationshipNode `json:"relationships,omitempty"`
	Properties    *ChangeBoardProperties        `json:"properties,omitempty"`
}

type ChangeBoard struct {
	ID            string                        `json:"id"`
	Name          string                        `json:"name"`
	Description   string                        `json:"description"`
	Filters       []ChangeBoardFilter           `json:"filters"`
	Groups        []ChangeBoardGroup            `json:"groups"`
	OwnerID       string                        `json:"owner_id"`
	OwnerType     string                        `json:"owner_type"`
	Relationships []ChangeBoardRelationshipNode `json:"relationships"`
	Properties    ChangeBoardProperties         `json:"properties"`
	CreatedAt     int64                         `json:"created_at"`
	UpdatedAt     int64                         `json:"updated_at"`
}

func (c *Client) CreateChangeBoard(req *ChangeBoardRequest) (*ChangeBoard, error) {
	var result ChangeBoard
	err := c.Post("/changeboards", req, &result)
	return &result, err
}

func (c *Client) GetChangeBoard(id string) (*ChangeBoard, error) {
	var result ChangeBoard
	err := c.Get(fmt.Sprintf("/changeboards/%s", id), &result)
	return &result, err
}

func (c *Client) UpdateChangeBoard(id string, req *ChangeBoardRequest) (*ChangeBoard, error) {
	var result ChangeBoard
	err := c.Put(fmt.Sprintf("/changeboards/%s", id), req, &result)
	return &result, err
}

func (c *Client) DeleteChangeBoard(id string) error {
	return c.Delete(fmt.Sprintf("/changeboards/%s", id))
}

// --- Alert snooze ---

type NodeSnoozeConfig struct {
	ID             string `json:"id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	Ident          string `json:"ident,omitempty"`
	Until          int    `json:"until"`
}

type AlertSnoozeDuration struct {
	AlertSnoozedUntil int `json:"alert_snoozed_until"`
}

func (c *Client) GetEntitySnooze(entityID string) (*AlertSnoozeDuration, error) {
	var result AlertSnoozeDuration
	err := c.Get(fmt.Sprintf("/entities/%s/snooze", entityID), &result)
	return &result, err
}

func (c *Client) UpsertEntitySnooze(entityID string, until int) (*NodeSnoozeConfig, error) {
	req := &NodeSnoozeConfig{Until: until}
	var result NodeSnoozeConfig
	err := c.Post(fmt.Sprintf("/entities/%s/alert-rules/snooze", entityID), req, &result)
	return &result, err
}

// --- OTel drop / forward (individual REST) ---

type OTelSettingFilter struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	Operator    string  `json:"operator"`
	Conjunction *string `json:"conjunction,omitempty"`
}

type OTelDropAction struct {
	Name        string            `json:"name"`
	Destination string            `json:"destination,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

type OTelDropProperties struct {
	Telemetry string             `json:"telemetry"`
	Filters   []OTelSettingFilter `json:"filters"`
	Action    OTelDropAction     `json:"action"`
}

type OTelDropRequest struct {
	Name       string             `json:"name"`
	Properties OTelDropProperties `json:"properties"`
}

type OTelDropResponse struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Properties OTelDropProperties `json:"properties"`
	CreatedAt  int64              `json:"created_at"`
	Status     string             `json:"status"`
}

func (c *Client) CreateOTelDrop(region, clusterID string, req *OTelDropRequest) (*OTelDropResponse, error) {
	var result OTelDropResponse
	path := fmt.Sprintf("/otel_settings/drop?region=%s&cluster_id=%s", region, clusterID)
	err := c.Post(path, req, &result)
	return &result, err
}

func (c *Client) GetOTelDrop(id, region string) (*OTelDropResponse, error) {
	var result OTelDropResponse
	err := c.Get(fmt.Sprintf("/otel_settings/drop/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) UpdateOTelDrop(id, region, clusterID string, req *OTelDropRequest) (*OTelDropResponse, error) {
	var result OTelDropResponse
	path := fmt.Sprintf("/otel_settings/drop/%s?region=%s&cluster_id=%s", id, region, clusterID)
	err := c.Put(path, req, &result)
	return &result, err
}

func (c *Client) DeleteOTelDrop(id, region, clusterID string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/drop/%s?region=%s&cluster_id=%s", id, region, clusterID))
}

type OTelForwardProperties struct {
	Telemetry   string              `json:"telemetry"`
	Filters     []OTelSettingFilter `json:"filters"`
	Destination string              `json:"destination"`
}

type OTelForwardRequest struct {
	Name       string                `json:"name"`
	Properties OTelForwardProperties `json:"properties"`
}

type OTelForwardResponse struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Properties OTelForwardProperties `json:"properties"`
	CreatedAt  int64                 `json:"created_at"`
	Status     string                `json:"status"`
}

func (c *Client) CreateOTelForward(region, clusterID string, req *OTelForwardRequest) (*OTelForwardResponse, error) {
	var result OTelForwardResponse
	path := fmt.Sprintf("/otel_settings/forward?region=%s&cluster_id=%s", region, clusterID)
	err := c.Post(path, req, &result)
	return &result, err
}

func (c *Client) GetOTelForward(id, region string) (*OTelForwardResponse, error) {
	var result OTelForwardResponse
	err := c.Get(fmt.Sprintf("/otel_settings/forward/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) UpdateOTelForward(id, region, clusterID string, req *OTelForwardRequest) (*OTelForwardResponse, error) {
	var result OTelForwardResponse
	path := fmt.Sprintf("/otel_settings/forward/%s?region=%s&cluster_id=%s", id, region, clusterID)
	err := c.Put(path, req, &result)
	return &result, err
}

func (c *Client) DeleteOTelForward(id, region, clusterID string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/forward/%s?region=%s&cluster_id=%s", id, region, clusterID))
}

// --- Sensitive data ---

type SensitiveDataScanRules struct {
	Email            bool `json:"email"`
	PhoneNumber      bool `json:"phone_number"`
	CreditCardNumber bool `json:"credit_card_number"`
}

type SensitiveDataAction struct {
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties,omitempty"`
}

type SensitiveDataProperties struct {
	Telemetry string                 `json:"telemetry"`
	ScanRules SensitiveDataScanRules `json:"scan_rules"`
	Action    SensitiveDataAction    `json:"action"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Order     int64                  `json:"order"`
}

type SensitiveDataRequest struct {
	Name       string                   `json:"name"`
	Properties SensitiveDataProperties `json:"properties"`
}

type SensitiveDataResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Properties SensitiveDataProperties `json:"properties"`
	CreatedAt  int64                   `json:"created_at"`
	Status     string                  `json:"status"`
}

func (c *Client) CreateSensitiveData(region string, req *SensitiveDataRequest) (*SensitiveDataResponse, error) {
	var result SensitiveDataResponse
	err := c.Post(fmt.Sprintf("/otel_settings/sensitive_data?region=%s", region), req, &result)
	return &result, err
}

func (c *Client) GetSensitiveData(id, region string) (*SensitiveDataResponse, error) {
	var result SensitiveDataResponse
	err := c.Get(fmt.Sprintf("/otel_settings/sensitive_data/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) UpdateSensitiveData(id, region string, req *SensitiveDataRequest) (*SensitiveDataResponse, error) {
	var result SensitiveDataResponse
	err := c.Put(fmt.Sprintf("/otel_settings/sensitive_data/%s?region=%s", id, region), req, &result)
	return &result, err
}

func (c *Client) DeleteSensitiveData(id, region string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/sensitive_data/%s?region=%s", id, region))
}

// --- Physical index (otel) ---

type PhysicalIndexProperties struct {
	Description     string              `json:"description,omitempty"`
	Telemetry       string              `json:"telemetry"`
	Filters         []OTelSettingFilter `json:"filters"`
	Destination     string              `json:"destination,omitempty"`
	BucketName      *string             `json:"bucket_name,omitempty"`
	Retain          bool                `json:"retain"`
	RetentionPeriod *int                `json:"retention_period,omitempty"`
}

type PhysicalIndexRequest struct {
	Name       string                   `json:"name"`
	Properties PhysicalIndexProperties `json:"properties"`
}

type PhysicalIndexResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Properties PhysicalIndexProperties `json:"properties"`
	CreatedAt  int64                   `json:"created_at"`
	Status     string                  `json:"status"`
}

func (c *Client) CreatePhysicalIndex(region, clusterID string, req *PhysicalIndexRequest) (*PhysicalIndexResponse, error) {
	var result PhysicalIndexResponse
	path := fmt.Sprintf("/otel_settings/physical_index?region=%s&cluster_id=%s", region, clusterID)
	err := c.Post(path, req, &result)
	return &result, err
}

func (c *Client) GetPhysicalIndex(id, region string) (*PhysicalIndexResponse, error) {
	var result PhysicalIndexResponse
	err := c.Get(fmt.Sprintf("/otel_settings/physical_index/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) UpdatePhysicalIndex(id, region, clusterID string, req *PhysicalIndexRequest) (*PhysicalIndexResponse, error) {
	var result PhysicalIndexResponse
	path := fmt.Sprintf("/otel_settings/physical_index/%s?region=%s&cluster_id=%s", id, region, clusterID)
	err := c.Put(path, req, &result)
	return &result, err
}

func (c *Client) DeletePhysicalIndex(id, region, clusterID string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/physical_index/%s?region=%s&cluster_id=%s", id, region, clusterID))
}

// --- Rehydration (otel) ---

type RehydrationOTelProperties struct {
	BucketName            *string             `json:"bucket_name,omitempty"`
	PhysicalIndex         string              `json:"physical_index"`
	Telemetry             string              `json:"telemetry"`
	Filters               []OTelSettingFilter `json:"filters,omitempty"`
	NotificationChannelID *int64              `json:"notification_channel_id,omitempty"`
	From                  int64               `json:"from"`
	To                    int64               `json:"to"`
	Message               string              `json:"message,omitempty"`
	Granularity           string              `json:"granularity,omitempty"`
	Targets               []string            `json:"targets,omitempty"`
}

type RehydrationOTelRequest struct {
	Name       string                    `json:"name"`
	Properties RehydrationOTelProperties `json:"properties"`
}

type RehydrationOTelResponse struct {
	ID         string                    `json:"id"`
	Name       string                    `json:"name"`
	Properties RehydrationOTelProperties `json:"properties"`
	CreatedAt  int64                     `json:"created_at"`
	Status     string                    `json:"status"`
}

func (c *Client) CreateOTelRehydration(region string, req *RehydrationOTelRequest) (*RehydrationOTelResponse, error) {
	var result RehydrationOTelResponse
	err := c.Post(fmt.Sprintf("/otel_settings/rehydration?region=%s", region), req, &result)
	return &result, err
}

func (c *Client) GetOTelRehydration(id, region string) (*RehydrationOTelResponse, error) {
	var result RehydrationOTelResponse
	err := c.Get(fmt.Sprintf("/otel_settings/rehydration/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) DeleteOTelRehydration(id, region string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/rehydration/%s?region=%s", id, region))
}

// --- Datasources ---

type Datasource struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Region  string `json:"region"`
	Default bool   `json:"default"`
}

func (c *Client) ListDatasources() ([]Datasource, error) {
	var result []Datasource
	err := c.Get("/datasources", &result)
	return result, err
}

// --- Users ---

type User struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	OrganizationID string  `json:"organization_id"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	DeletedAt      *int    `json:"deleted_at,omitempty"`
	CreatedAt      string  `json:"created_at,omitempty"`
	UpdatedAt      string  `json:"updated_at,omitempty"`
}

type InviteUsersRequest struct {
	Emails []string `json:"emails"`
	Role   string   `json:"role,omitempty"`
}

type PatchUserRequest struct {
	Active *bool `json:"active,omitempty"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

type UserRoleResponse struct {
	Role string `json:"role"`
}

func (c *Client) ListUsers() ([]User, error) {
	var result []User
	err := c.Get("/users", &result)
	return result, err
}

func (c *Client) InviteUsers(emails []string, role string) error {
	req := &InviteUsersRequest{Emails: emails, Role: role}
	return c.Post("/users/invite", req, nil)
}

func (c *Client) PatchUser(userID string, active bool) error {
	req := &PatchUserRequest{Active: &active}
	return c.Patch(fmt.Sprintf("/users/%s", userID), req, nil)
}

func (c *Client) DeleteUser(userID string) error {
	return c.Delete(fmt.Sprintf("/users/%s", userID))
}

func (c *Client) GetUserRole(userID string) (string, error) {
	var result UserRoleResponse
	err := c.Get(fmt.Sprintf("/users/%s/roles", userID), &result)
	return result.Role, err
}

func (c *Client) UpdateUserRole(userID, role string) error {
	return c.Put(fmt.Sprintf("/users/%s/roles", userID), &UpdateUserRoleRequest{Role: role}, nil)
}

func (c *Client) DeleteUserRole(userID string) error {
	return c.Delete(fmt.Sprintf("/users/%s/roles", userID))
}

func (c *Client) FindUserByEmail(email string) (*User, error) {
	users, err := c.ListUsers()
	if err != nil {
		return nil, err
	}
	want := strings.ToLower(strings.TrimSpace(email))
	for i := range users {
		if strings.ToLower(users[i].Email) == want {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user with email %q not found", email)
}

func (c *Client) FindUserByID(id string) (*User, error) {
	users, err := c.ListUsers()
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].ID == id {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user with id %q not found", id)
}

// --- Streaming aggregations ---

type StreamingAggProperties struct {
	Metric       string   `json:"metric"`
	Resolution   string   `json:"resolution"`
	Aggregation  string   `json:"aggregation"`
	Clause       string   `json:"clause"`
	Labels       []string `json:"labels"`
	OutputMetric string   `json:"output_metric"`
	WithName     string   `json:"with_name,omitempty"`
	WithValue    string   `json:"with_value,omitempty"`
}

type StreamingAggRequest struct {
	Name       string                 `json:"name"`
	Telemetry  string                 `json:"telemetry"`
	Properties StreamingAggProperties `json:"properties"`
}

type StreamingAggregation struct {
	ID        string                 `json:"id"`
	OrgID     string                 `json:"organization_id"`
	Region    string                 `json:"region"`
	ClusterID string                 `json:"cluster_id"`
	Telemetry string                 `json:"telemetry"`
	Name      string                 `json:"name"`
	Properties StreamingAggProperties `json:"properties"`
	CreatedAt int64                  `json:"created_at"`
	UpdatedAt int64                  `json:"updated_at"`
}

func sapRegionHeader(region string) map[string]string {
	return map[string]string{"region": region}
}

func (c *Client) CreateStreamingAggregation(clusterID, region string, req *StreamingAggRequest) (*StreamingAggregation, error) {
	var result StreamingAggregation
	path := fmt.Sprintf("/clusters/%s/streaming_aggregations", clusterID)
	err := c.PostWithHeaders(path, req, &result, sapRegionHeader(region))
	return &result, err
}

func (c *Client) ListStreamingAggregations(clusterID, region string) ([]StreamingAggregation, error) {
	var result []StreamingAggregation
	path := fmt.Sprintf("/clusters/%s/streaming_aggregations", clusterID)
	err := c.GetWithHeaders(path, &result, sapRegionHeader(region))
	return result, err
}

func (c *Client) GetStreamingAggregation(clusterID, region, id string) (*StreamingAggregation, error) {
	list, err := c.ListStreamingAggregations(clusterID, region)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i], nil
		}
	}
	return nil, fmt.Errorf("streaming aggregation %s not found", id)
}

func (c *Client) UpdateStreamingAggregation(clusterID, region, id string, req *StreamingAggRequest) (*StreamingAggregation, error) {
	var result StreamingAggregation
	path := fmt.Sprintf("/clusters/%s/streaming_aggregations/%s", clusterID, id)
	err := c.PutWithHeaders(path, req, &result, sapRegionHeader(region))
	return &result, err
}

func (c *Client) DeleteStreamingAggregation(clusterID, region, id string) error {
	path := fmt.Sprintf("/clusters/%s/streaming_aggregations/%s", clusterID, id)
	return c.DeleteWithHeaders(path, sapRegionHeader(region))
}

// --- Cold storage bucket ---

type ColdStorageBucketProperties struct {
	Default         bool   `json:"default"`
	AWSRegion       string `json:"aws_region"`
	AWSBucket       string `json:"aws_bucket"`
	AuthType        string `json:"auth_type"`
	AWSAccessKey    string `json:"aws_access_key,omitempty"`
	AWSSecretKey    string `json:"aws_secret_key,omitempty"`
	AWSRole         string `json:"aws_role,omitempty"`
	RetentionPeriod *int   `json:"retention_period,omitempty"`
}

type ColdStorageBucketRequest struct {
	Name       string                       `json:"name"`
	Properties ColdStorageBucketProperties  `json:"properties"`
}

type ColdStorageBucketResponse struct {
	ID         string                      `json:"id"`
	Name       string                      `json:"name"`
	Properties ColdStorageBucketProperties `json:"properties"`
	Status     string                      `json:"status"`
	CreatedAt  int64                       `json:"created_at"`
}

func (c *Client) CreateColdStorageBucket(region string, req *ColdStorageBucketRequest) (*ColdStorageBucketResponse, error) {
	var result ColdStorageBucketResponse
	err := c.Post(fmt.Sprintf("/otel_settings/cold_storage/bucket?region=%s", region), req, &result)
	return &result, err
}

func (c *Client) GetColdStorageBucket(id, region string) (*ColdStorageBucketResponse, error) {
	var result ColdStorageBucketResponse
	err := c.Get(fmt.Sprintf("/otel_settings/cold_storage/bucket/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) UpdateColdStorageBucket(id, region string, req *ColdStorageBucketRequest) (*ColdStorageBucketResponse, error) {
	var result ColdStorageBucketResponse
	err := c.Put(fmt.Sprintf("/otel_settings/cold_storage/bucket/%s?region=%s", id, region), req, &result)
	return &result, err
}

func (c *Client) DeleteColdStorageBucket(id, region string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/cold_storage/bucket/%s?region=%s", id, region))
}

func (c *Client) MarkDefaultColdStorageBucket(id, region string) error {
	return c.Patch(fmt.Sprintf("/otel_settings/cold_storage/bucket/mark_default/%s?region=%s", id, region), nil, nil)
}

// --- Cold storage backup ---

type ColdStorageBackupProperties struct {
	Enabled     *bool    `json:"enabled"`
	BucketName  string   `json:"bucket_name"`
	Granularity string   `json:"granularity"`
	Targets     []string `json:"targets,omitempty"`
}

type ColdStorageBackupRequest struct {
	Name       string                      `json:"name"`
	Properties ColdStorageBackupProperties `json:"properties"`
}

type ColdStorageBackupResponse struct {
	ID         string                      `json:"id"`
	Name       string                      `json:"name"`
	Properties ColdStorageBackupProperties `json:"properties"`
	Status     string                      `json:"status"`
	CreatedAt  int64                       `json:"created_at"`
}

func (c *Client) CreateColdStorageBackup(region string, req *ColdStorageBackupRequest) (*ColdStorageBackupResponse, error) {
	var result ColdStorageBackupResponse
	err := c.Post(fmt.Sprintf("/otel_settings/cold_storage/backup?region=%s", region), req, &result)
	return &result, err
}

func (c *Client) GetColdStorageBackup(id, region string) (*ColdStorageBackupResponse, error) {
	var result ColdStorageBackupResponse
	err := c.Get(fmt.Sprintf("/otel_settings/cold_storage/backup/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) UpdateColdStorageBackup(id, region string, req *ColdStorageBackupRequest) (*ColdStorageBackupResponse, error) {
	var result ColdStorageBackupResponse
	err := c.Put(fmt.Sprintf("/otel_settings/cold_storage/backup/%s?region=%s", id, region), req, &result)
	return &result, err
}

func (c *Client) DeleteColdStorageBackup(id, region string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/cold_storage/backup/%s?region=%s", id, region))
}

// --- S3 ingest ---

type S3IngestProperties struct {
	Default   bool   `json:"default"`
	AWSBucket string `json:"aws_bucket"`
	AuthType  string `json:"auth_type"`
	AWSRole   string `json:"aws_role"`
	AWSRegion string `json:"aws_region"`
}

type S3IngestRequest struct {
	Name       string            `json:"name"`
	Properties S3IngestProperties `json:"properties"`
}

type S3IngestResponse struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Properties S3IngestProperties `json:"properties"`
	Status     string            `json:"status"`
	CreatedAt  int64             `json:"created_at"`
}

func (c *Client) CreateS3Ingest(region string, req *S3IngestRequest) (*S3IngestResponse, error) {
	var result S3IngestResponse
	err := c.Post(fmt.Sprintf("/otel_settings/s3_ingest?region=%s", region), req, &result)
	return &result, err
}

func (c *Client) GetS3Ingest(id, region string) (*S3IngestResponse, error) {
	var result S3IngestResponse
	err := c.Get(fmt.Sprintf("/otel_settings/s3_ingest/%s?region=%s", id, region), &result)
	return &result, err
}

func (c *Client) UpdateS3Ingest(id, region string, req *S3IngestRequest) (*S3IngestResponse, error) {
	var result S3IngestResponse
	err := c.Put(fmt.Sprintf("/otel_settings/s3_ingest/%s?region=%s", id, region), req, &result)
	return &result, err
}

func (c *Client) DeleteS3Ingest(id, region string) error {
	return c.Delete(fmt.Sprintf("/otel_settings/s3_ingest/%s?region=%s", id, region))
}
