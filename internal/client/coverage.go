package client

import (
	"encoding/json"
	"fmt"
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
	var result SyntheticCheck
	err := c.Post("/synthetic/checks", req, &result)
	return &result, err
}

func (c *Client) GetSyntheticCheck(id string) (*SyntheticCheck, error) {
	var result SyntheticCheck
	err := c.Get(fmt.Sprintf("/synthetic/checks/%s", id), &result)
	return &result, err
}

func (c *Client) UpdateSyntheticCheck(id string, req *UpdateSyntheticCheckRequest) (*SyntheticCheck, error) {
	var result SyntheticCheck
	err := c.Put(fmt.Sprintf("/synthetic/checks/%s", id), req, &result)
	return &result, err
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
