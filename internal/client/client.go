package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	// Direct access token (legacy) - for read/write operations
	APIToken string
	// Refresh token for automatic token management
	RefreshToken string
	// Delete token - required for delete operations (separate scope)
	DeleteToken string
	// Refresh token for delete operations (generates access tokens with delete scope)
	DeleteRefreshToken string
	// Organization slug
	Org string
	// API base URL
	BaseURL string
}

type AccessToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    int64     `json:"expires_at"`
	IssuedAt     int64     `json:"issued_at"`
	Type         string    `json:"type"`
	Scopes       []string  `json:"scopes"`
	expiresAt    time.Time // Cached parsed time
}

type Client struct {
	config     *Config
	httpClient *http.Client
	// Token management for read/write operations
	tokenMutex  sync.RWMutex
	accessToken *AccessToken
	// Delete token management (separate from access token due to scope requirements)
	deleteToken       string       // Static delete token (legacy)
	deleteAccessToken *AccessToken // Cached delete access token from refresh
	deleteTokenMutex  sync.RWMutex // Mutex for delete token refresh
}

func NewClient(config *Config) (*Client, error) {
	client := &Client{
		config:      config,
		deleteToken: config.DeleteToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// If refresh token is provided, fetch initial access token
	if config.RefreshToken != "" {
		token, err := client.refreshAccessToken()
		if err != nil {
			return nil, fmt.Errorf("failed to get initial access token: %w", err)
		}
		client.accessToken = token
	} else if config.APIToken != "" {
		// Use direct access token (legacy mode)
		client.accessToken = &AccessToken{
			AccessToken: config.APIToken,
			expiresAt:   time.Now().Add(24 * time.Hour), // Assume valid for 24h
		}
	} else {
		return nil, fmt.Errorf("either api_token or refresh_token must be provided")
	}

	return client, nil
}

func (c *Client) getAccessToken() (string, error) {
	c.tokenMutex.RLock()
	token := c.accessToken
	c.tokenMutex.RUnlock()

	// Check if token is expired or will expire soon (within 5 minutes)
	if token != nil && time.Now().Before(token.expiresAt.Add(-5*time.Minute)) {
		return token.AccessToken, nil
	}

	// Token expired or missing, refresh it
	if c.config.RefreshToken != "" {
		c.tokenMutex.Lock()
		defer c.tokenMutex.Unlock()

		// Double-check after acquiring lock
		if c.accessToken != nil && time.Now().Before(c.accessToken.expiresAt.Add(-5*time.Minute)) {
			return c.accessToken.AccessToken, nil
		}

		newToken, err := c.refreshAccessToken()
		if err != nil {
			return "", fmt.Errorf("failed to refresh access token: %w", err)
		}
		c.accessToken = newToken
		return newToken.AccessToken, nil
	}

	// Fallback to direct API token
	if c.config.APIToken != "" {
		return c.config.APIToken, nil
	}

	return "", fmt.Errorf("no valid authentication token available")
}

func (c *Client) refreshAccessToken() (*AccessToken, error) {
	reqBody := struct {
		RefreshToken string `json:"refresh_token"`
	}{
		RefreshToken: c.config.RefreshToken,
	}

	reqURL := fmt.Sprintf("%s/api/v4/oauth/access_token", c.config.BaseURL)
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var token AccessToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse expires_at
	token.expiresAt = time.Unix(token.ExpiresAt, 0)

	return &token, nil
}

// refreshDeleteAccessToken obtains a new access token using the delete refresh token.
// The delete refresh token generates access tokens with delete scope.
func (c *Client) refreshDeleteAccessToken() (*AccessToken, error) {
	reqBody := struct {
		RefreshToken string `json:"refresh_token"`
	}{
		RefreshToken: c.config.DeleteRefreshToken,
	}

	reqURL := fmt.Sprintf("%s/api/v4/oauth/access_token", c.config.BaseURL)
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var token AccessToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse expires_at
	token.expiresAt = time.Unix(token.ExpiresAt, 0)

	return &token, nil
}

// getDeleteAccessToken returns a valid delete access token, refreshing if necessary.
// Falls back to static delete token if no delete refresh token is configured.
func (c *Client) getDeleteAccessToken() (string, error) {
	// If delete refresh token is configured, use dynamic token management
	if c.config.DeleteRefreshToken != "" {
		c.deleteTokenMutex.RLock()
		token := c.deleteAccessToken
		c.deleteTokenMutex.RUnlock()

		// Check if token is valid and not expiring soon
		if token != nil && time.Now().Before(token.expiresAt.Add(-5*time.Minute)) {
			return token.AccessToken, nil
		}

		// Token expired or missing, refresh it
		c.deleteTokenMutex.Lock()
		defer c.deleteTokenMutex.Unlock()

		// Double-check after acquiring lock
		if c.deleteAccessToken != nil && time.Now().Before(c.deleteAccessToken.expiresAt.Add(-5*time.Minute)) {
			return c.deleteAccessToken.AccessToken, nil
		}

		newToken, err := c.refreshDeleteAccessToken()
		if err != nil {
			return "", fmt.Errorf("failed to refresh delete access token: %w", err)
		}
		c.deleteAccessToken = newToken
		return newToken.AccessToken, nil
	}

	// Fall back to static delete token (legacy mode)
	if c.deleteToken != "" {
		return c.deleteToken, nil
	}

	return "", fmt.Errorf("delete_token or delete_refresh_token is required for delete operations")
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	// Get valid access token
	accessToken, err := c.getAccessToken()
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	reqURL := fmt.Sprintf("%s/api/v4/organizations/%s%s", c.config.BaseURL, c.config.Org, path)
	req, err := http.NewRequest(method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use X-LAST9-API-TOKEN header with Bearer prefix as per Last9 API docs
	req.Header.Set("X-LAST9-API-TOKEN", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return resp, nil
}

func (c *Client) Get(path string, result interface{}) error {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) Post(path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) Put(path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest("PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) Patch(path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest("PATCH", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) Delete(path string) error {
	// Get valid delete access token (handles refresh token or static token)
	deleteToken, err := c.getDeleteAccessToken()
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s/api/v4/organizations/%s%s", c.config.BaseURL, c.config.Org, path)
	req, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Use delete token for delete operations
	req.Header.Set("X-LAST9-API-TOKEN", fmt.Sprintf("Bearer %s", deleteToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// Dashboard methods
func (c *Client) GetDashboard(id string) (*Dashboard, error) {
	var dashboard Dashboard
	err := c.Get(fmt.Sprintf("/dashboards/%s", id), &dashboard)
	return &dashboard, err
}

func (c *Client) CreateDashboard(dashboard *DashboardCreateRequest) (*Dashboard, error) {
	var result Dashboard
	err := c.Post("/dashboards", dashboard, &result)
	return &result, err
}

func (c *Client) UpdateDashboard(id string, dashboard *DashboardUpdateRequest) (*Dashboard, error) {
	var result Dashboard
	err := c.Put(fmt.Sprintf("/dashboards/%s", id), dashboard, &result)
	return &result, err
}

func (c *Client) DeleteDashboard(id string) error {
	return c.Delete(fmt.Sprintf("/dashboards/%s", id))
}

// Alert methods
func (c *Client) GetAlert(entityID, alertID string) (*Alert, error) {
	var alert Alert
	err := c.Get(fmt.Sprintf("/entities/%s/alert-rules/%s", entityID, alertID), &alert)
	return &alert, err
}

func (c *Client) CreateAlert(entityID string, alert *AlertCreateRequest) (*Alert, error) {
	var result Alert
	err := c.Post(fmt.Sprintf("/entities/%s/alert-rules", entityID), alert, &result)
	return &result, err
}

func (c *Client) UpdateAlert(entityID, alertID string, alert *AlertUpdateRequest) (*Alert, error) {
	var result Alert
	err := c.Put(fmt.Sprintf("/entities/%s/alert-rules/%s", entityID, alertID), alert, &result)
	return &result, err
}

func (c *Client) DeleteAlert(entityID, alertID string) error {
	return c.Delete(fmt.Sprintf("/entities/%s/alert-rules/%s", entityID, alertID))
}

// Macro methods
func (c *Client) GetMacro(clusterID string) (*Macro, error) {
	var macro Macro
	err := c.Get(fmt.Sprintf("/clusters/%s/macros", clusterID), &macro)
	return &macro, err
}

func (c *Client) UpsertMacro(clusterID string, macro *MacroUpsertRequest) (*Macro, error) {
	var result Macro
	err := c.Post(fmt.Sprintf("/clusters/%s/macros", clusterID), macro, &result)
	return &result, err
}

func (c *Client) DeleteMacro(clusterID string) error {
	return c.Delete(fmt.Sprintf("/clusters/%s/macros", clusterID))
}

// Cluster represents a Last9 cluster
type Cluster struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Region    string `json:"region"`
	IsDefault bool   `json:"default"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ClustersResponse wraps the API response for clusters
type ClustersResponse struct {
	Data   []Cluster `json:"data"`
	Status string    `json:"status"`
}

// GetClusters fetches all clusters for a region
func (c *Client) GetClusters(region string) ([]Cluster, error) {
	var response ClustersResponse
	err := c.Get(fmt.Sprintf("/clusters?region=%s", region), &response)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

// GetDefaultCluster returns the default cluster for a region.
// If no default is found, returns the first cluster.
// Returns an error if no clusters exist.
func (c *Client) GetDefaultCluster(region string) (*Cluster, error) {
	clusters, err := c.GetClusters(region)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch clusters: %w", err)
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters found in region %s", region)
	}

	// Find the default cluster
	for i := range clusters {
		if clusters[i].IsDefault {
			return &clusters[i], nil
		}
	}

	// If no default is marked, return the first cluster
	return &clusters[0], nil
}

// Policy methods
func (c *Client) GetPolicy(id string) (*Policy, error) {
	var policy Policy
	err := c.Get(fmt.Sprintf("/policies/%s", id), &policy)
	return &policy, err
}

func (c *Client) CreatePolicy(policy *PolicyCreateRequest) (*Policy, error) {
	var result Policy
	err := c.Post("/policies", policy, &result)
	return &result, err
}

func (c *Client) UpdatePolicy(id string, policy *PolicyUpdateRequest) (*Policy, error) {
	var result Policy
	err := c.Patch(fmt.Sprintf("/policies/%s", id), policy, &result)
	return &result, err
}

func (c *Client) DeletePolicy(id string) error {
	return c.Delete(fmt.Sprintf("/policies/%s", id))
}

// Types
type Dashboard struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Readonly    bool             `json:"readonly"`
	Panels      []DashboardPanel `json:"panels"`
	Tags        []string         `json:"tags"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

type DashboardPanel struct {
	ID            string                 `json:"id,omitempty"`
	Title         string                 `json:"title"`
	Query         string                 `json:"query"`
	Visualization string                 `json:"visualization"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

type DashboardCreateRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Readonly    bool             `json:"readonly"`
	Panels      []DashboardPanel `json:"panels"`
	Tags        []string         `json:"tags"`
}

type DashboardUpdateRequest struct {
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Readonly    *bool            `json:"readonly,omitempty"`
	Panels      []DashboardPanel `json:"panels,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
}

type Alert struct {
	ID                           string          `json:"id"`
	Name                         string          `json:"rule_name"`
	Description                  string          `json:"description"`
	EntityID                     string          `json:"entity_id"`
	Indicator                    string          `json:"primary_indicator"`
	Expression                   string          `json:"expression,omitempty"`
	Condition                    string          `json:"condition,omitempty"`
	EvalWindow                   int             `json:"eval_window,omitempty"`
	AlertCondition               string          `json:"alert_condition,omitempty"`
	Severity                     string          `json:"severity"`
	MuteUntil                    int             `json:"mute_until"`
	IsDisabled                   bool            `json:"is_disabled"`
	Properties                   AlertProperties `json:"properties"`
	GroupTimeseriesNotifications bool            `json:"group_timeseries_notifications"`
	NotificationChannels         []string        `json:"notification_channels,omitempty"`
}

type AlertProperties struct {
	Description string                 `json:"description"`
	Runbook     map[string]interface{} `json:"runbook,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
}

type AlertCreateRequest struct {
	RuleName                     string                 `json:"rule_name"`
	PrimaryIndicator             string                 `json:"primary_indicator"`
	Expression                   string                 `json:"expression,omitempty"`
	Condition                    string                 `json:"condition,omitempty"`
	EvalWindow                   int                    `json:"eval_window,omitempty"`
	AlertCondition               string                 `json:"alert_condition,omitempty"`
	Severity                     string                 `json:"severity"`
	IsDisabled                   bool                   `json:"is_disabled"`
	Properties                   AlertProperties        `json:"properties"`
	GroupTimeseriesNotifications bool                   `json:"group_timeseries_notifications"`
	MuteUntil                    int                    `json:"mute_until"`
	ExpressionArgs               map[string]interface{} `json:"expression_args"`
	NotificationChannels         []string               `json:"notification_channels,omitempty"`
}

type AlertUpdateRequest struct {
	RuleName                     *string                `json:"rule_name,omitempty"`
	PrimaryIndicator             *string                `json:"primary_indicator,omitempty"`
	Expression                   *string                `json:"expression,omitempty"`
	Condition                    *string                `json:"condition,omitempty"`
	EvalWindow                   *int                   `json:"eval_window,omitempty"`
	AlertCondition               *string                `json:"alert_condition,omitempty"`
	Severity                     *string                `json:"severity,omitempty"`
	IsDisabled                   *bool                  `json:"is_disabled,omitempty"`
	Properties                   *AlertProperties       `json:"properties,omitempty"`
	GroupTimeseriesNotifications *bool                  `json:"group_timeseries_notifications,omitempty"`
	MuteUntil                    *int                   `json:"mute_until,omitempty"`
	ExpressionArgs               map[string]interface{} `json:"expression_args,omitempty"`
	NotificationChannels         []string               `json:"notification_channels,omitempty"`
}

type Macro struct {
	ID        string `json:"id"`
	ClusterID string `json:"cluster_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type MacroUpsertRequest struct {
	Body string `json:"body"`
}

type Policy struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Rules                []PolicyRule           `json:"rules"`
	Filters              map[string]interface{} `json:"filters"`
	EntityCount          int                    `json:"entity_count"`
	EntityCompliantCount int                    `json:"entity_compliant_count"`
}

type PolicyRule struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

type PolicyCreateRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       []PolicyRule           `json:"rules"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

type PolicyUpdateRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Rules       []PolicyRule           `json:"rules,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

// Control Plane methods
func (c *Client) GetDropRules(region string) (*DropRulesResponse, error) {
	var result DropRulesResponse
	err := c.Get(fmt.Sprintf("/logs_settings/routing?region=%s", region), &result)
	return &result, err
}

func (c *Client) UpsertDropRule(region string, rule *DropRule) (*DropRulesResponse, error) {
	var result DropRulesResponse
	err := c.Put(fmt.Sprintf("/logs_settings/routing?region=%s", region), rule, &result)
	return &result, err
}

func (c *Client) CreateDropRules(region string, rules *DropRulesRequest) (*DropRulesResponse, error) {
	var result DropRulesResponse
	err := c.Post(fmt.Sprintf("/logs_settings/routing?region=%s", region), rules, &result)
	return &result, err
}

func (c *Client) UpdateDropRules(region, clusterID string, rules *DropRulesRequest) (*DropRulesResponse, error) {
	var result DropRulesResponse
	err := c.Post(fmt.Sprintf("/logs_settings/routing?region=%s&cluster_id=%s", region, clusterID), rules, &result)
	return &result, err
}

func (c *Client) GetForwardRules(region string) (*ForwardRulesResponse, error) {
	var result ForwardRulesResponse
	err := c.Get(fmt.Sprintf("/logs_settings/forward?region=%s", region), &result)
	return &result, err
}

func (c *Client) UpsertForwardRules(region string, rules *ForwardRulesRequest) (*ForwardRulesResponse, error) {
	var result ForwardRulesResponse
	err := c.Post(fmt.Sprintf("/logs_settings/forward?region=%s", region), rules, &result)
	return &result, err
}

func (c *Client) UpdateForwardRules(region, clusterID string, rules *ForwardRulesRequest) (*ForwardRulesResponse, error) {
	var result ForwardRulesResponse
	err := c.Post(fmt.Sprintf("/logs_settings/forward?region=%s&cluster_id=%s", region, clusterID), rules, &result)
	return &result, err
}

func (c *Client) GetRehydration(region string) (*RehydrationResponse, error) {
	var result RehydrationResponse
	err := c.Get(fmt.Sprintf("/logs_settings/rehydration?region=%s", region), &result)
	return &result, err
}

func (c *Client) UpsertRehydration(region string, rehydration *RehydrationRequest) (*RehydrationResponse, error) {
	var result RehydrationResponse
	err := c.Post(fmt.Sprintf("/logs_settings/rehydration?region=%s", region), rehydration, &result)
	return &result, err
}

func (c *Client) UpdateRehydrationStatus(region string, status *RehydrationStatusRequest) error {
	return c.Put(fmt.Sprintf("/logs_settings/rehydration/status?region=%s", region), status, nil)
}

func (c *Client) GetPartitionConfigs(region string) (*PartitionConfigResponse, error) {
	var result PartitionConfigResponse
	err := c.Get(fmt.Sprintf("/logs_settings/physical_indexes?region=%s", region), &result)
	return &result, err
}

func (c *Client) UpsertPartitionConfigs(region string, configs *PartitionConfigRequest) (*PartitionConfigResponse, error) {
	var result PartitionConfigResponse
	err := c.Post(fmt.Sprintf("/logs_settings/physical_indexes?region=%s", region), configs, &result)
	return &result, err
}

// Control Plane Types
type DropRule struct {
	Name      string          `json:"name"`
	Telemetry string          `json:"telemetry"`
	Filters   []RoutingFilter `json:"filters"`
	Action    RoutingAction   `json:"action"`
}

type RoutingFilter struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	Operator    string  `json:"operator"`
	Conjunction *string `json:"conjunction,omitempty"`
}

type RoutingAction struct {
	Name        string            `json:"name"`
	Destination string            `json:"destination"`
	Properties  map[string]string `json:"properties"`
}

type DropRulesRequest struct {
	Properties []DropRule `json:"properties"`
}

type DropRulesResponse struct {
	ID         string     `json:"id"`
	Region     string     `json:"region"`
	Properties []DropRule `json:"properties"`
	CreatedAt  int64      `json:"created_at"`
	UpdatedAt  int64      `json:"updated_at"`
}

type ForwardRule struct {
	Name        string          `json:"name"`
	Telemetry   string          `json:"telemetry"`
	Filters     []RoutingFilter `json:"filters"`
	Destination string          `json:"destination"`
}

type ForwardRulesRequest struct {
	Properties []ForwardRule `json:"properties"`
}

type ForwardRulesResponse struct {
	ID         string        `json:"id"`
	Region     string        `json:"region"`
	Properties []ForwardRule `json:"properties"`
	CreatedAt  int64         `json:"created_at"`
	UpdatedAt  int64         `json:"updated_at"`
}

type RehydrationBlock struct {
	ID                    string          `json:"id"`
	BlockName             string          `json:"block_name"`
	BucketName            *string         `json:"bucket_name,omitempty"`
	PhysicalIndex         string          `json:"physical_index"`
	Telemetry             string          `json:"telemetry"`
	Filters               []RoutingFilter `json:"filters"`
	NotificationChannelID *int64          `json:"notification_channel_id,omitempty"`
	From                  int64           `json:"from"`
	To                    int64           `json:"to"`
	Status                string          `json:"status"`
	Message               string          `json:"message"`
	Granularity           string          `json:"granularity"`
	Targets               []string        `json:"targets"`
}

type RehydrationRequest struct {
	Properties []RehydrationBlock `json:"properties"`
}

type RehydrationResponse struct {
	ID         string             `json:"id"`
	Region     string             `json:"region"`
	Properties []RehydrationBlock `json:"properties"`
	CreatedAt  int64              `json:"created_at"`
	UpdatedAt  int64              `json:"updated_at"`
}

type RehydrationStatusRequest struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Region    string `json:"region"`
	BlockName string `json:"block_name"`
	Message   string `json:"message"`
}

type PartitionConfig struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Telemetry       string          `json:"telemetry"`
	Filters         []RoutingFilter `json:"filters"`
	Destination     string          `json:"destination"`
	BucketName      *string         `json:"bucket_name,omitempty"`
	RetentionPeriod *int            `json:"retention_period,omitempty"`
	Retain          bool            `json:"retain"`
	Status          string          `json:"status"`
}

type PartitionConfigRequest struct {
	Properties []PartitionConfig `json:"properties"`
}

type PartitionConfigResponse struct {
	ID         string            `json:"id"`
	Region     string            `json:"region"`
	Properties []PartitionConfig `json:"properties"`
	CreatedAt  int64             `json:"created_at"`
	UpdatedAt  int64             `json:"updated_at"`
}

// Notification Destination Types
type NotificationDestination struct {
	ID             int                    `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	Name           string                 `json:"name"`
	Destination    string                 `json:"destination"`
	Namespace      string                 `json:"namespace"`
	Service        string                 `json:"service"`
	Type           string                 `json:"type"` // email, slack, pagerduty, webhook, etc.
	SnoozeUntil    int64                  `json:"snooze_until"`
	Global         bool                   `json:"global"`
	Priority       int                    `json:"priority"`
	Property       map[string]interface{} `json:"property"`
	InUse          bool                   `json:"in_use"`
	Usage          map[string]int         `json:"usage"`
	ServiceFqid    string                 `json:"service_fqid"`
	Severity       string                 `json:"severity"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
	Services       []string               `json:"services"`
	SendResolved   bool                   `json:"send_resolved"`
}

// NotificationDestinationsResponse is kept for backward compatibility but the API returns an array directly

// Scheduled Search Alert Types
type ScheduledSearchAlert struct {
	RuleName      string                    `json:"rule_name"`
	QueryType     string                    `json:"query_type"`
	PhysicalIndex string                    `json:"physical_index"`
	RuleType      string                    `json:"rule_type"`
	Properties    ScheduledSearchProperties `json:"properties"`
}

type ScheduledSearchProperties struct {
	Telemetry         string                    `json:"telemetry"`
	Query             string                    `json:"query"` // JSON encoded pipeline
	SavedSearchID     string                    `json:"saved_search_id,omitempty"`
	PostProcessor     []PostProcessor           `json:"post_processor"`
	ResultantQuery    string                    `json:"resultant_query,omitempty"` // Computed by server
	SearchFrequency   int                       `json:"search_frequency"`
	AlertDestinations []NotificationDestination `json:"alert_destinations"`
	MetricName        string                    `json:"metric_name,omitempty"`
	Threshold         Threshold                 `json:"threshold"`
}

type PostProcessor struct {
	Type       string                 `json:"type"`
	Aggregates []Aggregate            `json:"aggregates"`
	Groupby    map[string]interface{} `json:"groupby"`
}

type Aggregate struct {
	Function map[string]interface{} `json:"function"`
	As       string                 `json:"as"`
}

type Threshold struct {
	Value    float64 `json:"value"`
	Operator string  `json:"operator"` // >, <, >=, <=, ==, !=
}

type ScheduledSearchRequest struct {
	Properties []ScheduledSearchAlert `json:"properties"`
}

// ScheduledSearchAlertFull represents the full alert object returned by the API
type ScheduledSearchAlertFull struct {
	ID             string                    `json:"id"`
	RuleName       string                    `json:"rule_name"`
	EntityID       string                    `json:"entity_id"`
	RuleType       string                    `json:"rule_type"`
	QueryType      string                    `json:"query_type"`
	PhysicalIndex  string                    `json:"physical_index"`
	Region         string                    `json:"region"`
	OrganizationID string                    `json:"organization_id"`
	Properties     ScheduledSearchProperties `json:"properties"`
	CreatedBy      string                    `json:"created_by"`
	CreatedAt      int64                     `json:"created_at"`
	UpdatedAt      int64                     `json:"updated_at"`
}

// ToScheduledSearchAlert converts the full API response to the request format
func (f *ScheduledSearchAlertFull) ToScheduledSearchAlert() ScheduledSearchAlert {
	return ScheduledSearchAlert{
		RuleName:      f.RuleName,
		QueryType:     f.QueryType,
		PhysicalIndex: f.PhysicalIndex,
		RuleType:      f.RuleType,
		Properties:    f.Properties,
	}
}

// Notification Destination methods
func (c *Client) ListNotificationDestinations() ([]NotificationDestination, error) {
	var result []NotificationDestination
	err := c.Get("/notification_settings", &result)
	return result, err
}

func (c *Client) GetNotificationDestination(id int) (*NotificationDestination, error) {
	destinations, err := c.ListNotificationDestinations()
	if err != nil {
		return nil, err
	}

	// Find destination by ID
	for _, dest := range destinations {
		if dest.ID == id {
			return &dest, nil
		}
	}

	return nil, fmt.Errorf("notification destination with ID %d not found", id)
}

// Scheduled Search methods
func (c *Client) GetScheduledSearchAlerts(region string) ([]ScheduledSearchAlertFull, error) {
	var result []ScheduledSearchAlertFull
	err := c.Get(fmt.Sprintf("/logs_settings/scheduled_search?region=%s", region), &result)
	return result, err
}

func (c *Client) CreateScheduledSearchAlert(region string, alert *ScheduledSearchAlert) (*ScheduledSearchAlertFull, error) {
	// The scheduled search API expects a single alert object, not an array
	// Each POST creates a new alert
	var result ScheduledSearchAlertFull
	err := c.Post(fmt.Sprintf("/logs_settings/scheduled_search?region=%s", region), alert, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateScheduledSearchAlert(region, alertID string, alert *ScheduledSearchAlert) (*ScheduledSearchAlertFull, error) {
	// The scheduled search API expects a single alert object for PUT updates
	var result ScheduledSearchAlertFull
	err := c.Put(fmt.Sprintf("/logs_settings/scheduled_search/%s?region=%s", alertID, region), alert, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteScheduledSearchAlert(region, alertID string) error {
	// Delete individual scheduled search alert by ID
	return c.Delete(fmt.Sprintf("/logs_settings/scheduled_search/%s?region=%s", alertID, region))
}

// Entity Types

// EntityMetadata contains metadata fields returned nested in the API response
type EntityMetadata struct {
	ID          string            `json:"id,omitempty"`
	EntityID    string            `json:"entity_id,omitempty"`
	Team        string            `json:"team,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Links       []EntityLink      `json:"links,omitempty"`
	AdhocFilter *AdhocFilter      `json:"adhoc_filter,omitempty"`
}

type Entity struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Type                 string            `json:"type"`
	ExternalRef          string            `json:"external_ref"`
	Description          string            `json:"description"`
	DataSource           string            `json:"data_source,omitempty"`
	DataSourceID         string            `json:"data_source_id,omitempty"`
	Namespace            string            `json:"namespace,omitempty"`
	Team                 string            `json:"team,omitempty"`
	Tier                 string            `json:"tier,omitempty"`
	Workspace            string            `json:"workspace,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	EntityClass          string            `json:"entity_class,omitempty"`
	UIReadonly           bool              `json:"ui_readonly"`
	AdhocFilter          *AdhocFilter      `json:"adhoc_filter,omitempty"`
	Indicators           []Indicator       `json:"indicators,omitempty"`
	Links                []EntityLink      `json:"links,omitempty"`
	NotificationChannels []string          `json:"notification_channels,omitempty"`
	CreatedAt            int64             `json:"created_at,omitempty"`
	UpdatedAt            int64             `json:"updated_at,omitempty"`
	// Metadata is returned nested in GET response - we extract fields from it
	Metadata *EntityMetadata `json:"metadata,omitempty"`
}

type AdhocFilter struct {
	DataSource string            `json:"data_source"`
	Labels     map[string]string `json:"labels"`
}

type Indicator struct {
	Name  string `json:"name"`
	Query string `json:"query"`
	Unit  string `json:"unit,omitempty"`
}

type EntityLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EntityCreateRequest struct {
	Name                 string            `json:"name"`
	Type                 string            `json:"type"`
	ExternalRef          string            `json:"external_ref"`
	Description          string            `json:"description,omitempty"`
	DataSource           string            `json:"data_source,omitempty"`
	DataSourceID         string            `json:"data_source_id,omitempty"`
	Namespace            string            `json:"namespace,omitempty"`
	Team                 string            `json:"team,omitempty"`
	Tier                 string            `json:"tier,omitempty"`
	Workspace            string            `json:"workspace,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	EntityClass          string            `json:"entity_class,omitempty"`
	UIReadonly           bool              `json:"ui_readonly"`
	AdhocFilter          *AdhocFilter      `json:"adhoc_filter,omitempty"`
	Indicators           []Indicator       `json:"indicators,omitempty"`
	Links                []EntityLink      `json:"links,omitempty"`
	NotificationChannels []string          `json:"notification_channels,omitempty"`
}

type EntityUpdateRequest struct {
	Name                 *string           `json:"name,omitempty"`
	Type                 *string           `json:"type,omitempty"`
	ExternalRef          *string           `json:"external_ref,omitempty"`
	Description          *string           `json:"description,omitempty"`
	DataSource           *string           `json:"data_source,omitempty"`
	DataSourceID         *string           `json:"data_source_id,omitempty"`
	Namespace            *string           `json:"namespace,omitempty"`
	Team                 *string           `json:"team,omitempty"`
	Tier                 *string           `json:"tier,omitempty"`
	Workspace            *string           `json:"workspace,omitempty"`
	Tags                 []string          `json:"tags,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	EntityClass          *string           `json:"entity_class,omitempty"`
	UIReadonly           *bool             `json:"ui_readonly,omitempty"`
	AdhocFilter          *AdhocFilter      `json:"adhoc_filter,omitempty"`
	Indicators           []Indicator       `json:"indicators,omitempty"`
	Links                []EntityLink      `json:"links,omitempty"`
	NotificationChannels []string          `json:"notification_channels,omitempty"`
}

type EntitiesListResponse struct {
	Entities []Entity `json:"entities"`
}

// Entity methods
func (c *Client) GetEntity(id string) (*Entity, error) {
	var entity Entity
	err := c.Get(fmt.Sprintf("/entities/%s", id), &entity)
	return &entity, err
}

func (c *Client) GetEntityByExternalRef(externalRef string) (*Entity, error) {
	var response EntitiesListResponse
	err := c.Get(fmt.Sprintf("/entities?external_ref=%s", externalRef), &response)
	if err != nil {
		return nil, err
	}
	if len(response.Entities) == 0 {
		return nil, fmt.Errorf("entity with external_ref '%s' not found", externalRef)
	}
	return &response.Entities[0], nil
}

func (c *Client) ListEntities() (*EntitiesListResponse, error) {
	var response EntitiesListResponse
	err := c.Get("/entities", &response)
	return &response, err
}

func (c *Client) CreateEntity(entity *EntityCreateRequest) (*Entity, error) {
	var result Entity
	err := c.Post("/entities", entity, &result)
	return &result, err
}

func (c *Client) UpdateEntity(id string, entity *EntityUpdateRequest) (*Entity, error) {
	var result Entity
	err := c.Put(fmt.Sprintf("/entities/%s", id), entity, &result)
	return &result, err
}

func (c *Client) DeleteEntity(id string) error {
	return c.Delete(fmt.Sprintf("/entities/%s", id))
}

// EntityMetadataUpdateRequest for updating entity metadata (tags, labels, team, links)
// This is a separate API endpoint from entity update
type EntityMetadataUpdateRequest struct {
	Team        string            `json:"team"`
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Links       []EntityLink      `json:"links,omitempty"`
	AdhocFilter *AdhocFilter      `json:"adhoc_filter,omitempty"`
}

func (c *Client) UpdateEntityMetadata(entityID string, metadata *EntityMetadataUpdateRequest) error {
	var result map[string]interface{}
	return c.Put(fmt.Sprintf("/entities/%s/metadata", entityID), metadata, &result)
}

// KPI Types
type KPI struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Definition     KPIDefinition `json:"definition"`
	KPIType        string        `json:"kpi_type"`
	OrganizationID string        `json:"organization_id"`
	EntityID       string        `json:"entity_id"`
	CreatedAt      int64         `json:"created_at"`
	UpdatedAt      int64         `json:"updated_at"`
}

type KPIDefinition struct {
	Query  string `json:"query"`
	Source string `json:"source"`
	Unit   string `json:"unit"`
}

type KPICreateRequest struct {
	Name       string        `json:"name"`
	Definition KPIDefinition `json:"definition"`
	KPIType    string        `json:"kpi_type"`
}

type KPIUpdateRequest struct {
	Name       string        `json:"name"`
	Definition KPIDefinition `json:"definition"`
	KPIType    string        `json:"kpi_type"`
}

// NotificationChannel CRUD methods

// NotificationChannelRequest represents the request body for creating/updating a notification channel
type NotificationChannelRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Destination  string `json:"destination"`
	SendResolved bool   `json:"send_resolved"`
}

// CreateNotificationDestination creates a new notification channel (master channel)
func (c *Client) CreateNotificationDestination(req *NotificationChannelRequest) (*NotificationDestination, error) {
	var result NotificationDestination
	err := c.Post("/notification_settings", req, &result)
	return &result, err
}

// UpdateNotificationDestination updates an existing notification channel
func (c *Client) UpdateNotificationDestination(id int, req *NotificationChannelRequest) (*NotificationDestination, error) {
	var result NotificationDestination
	err := c.Put(fmt.Sprintf("/notification_settings/%d", id), req, &result)
	return &result, err
}

// DeleteNotificationDestination deletes a notification channel
// Note: This uses a different endpoint format than other operations:
// /api/organizations/{org}/workspace/{org}/notification_settings/{id} (no /v4 prefix)
func (c *Client) DeleteNotificationDestination(id int) error {
	// Get valid delete access token
	deleteToken, err := c.getDeleteAccessToken()
	if err != nil {
		return err
	}

	// Use the workspace-based endpoint (no /v4 prefix)
	reqURL := fmt.Sprintf("%s/api/organizations/%s/workspace/%s/notification_settings/%d",
		c.config.BaseURL, c.config.Org, c.config.Org, id)

	req, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Use X-LAST9-API-TOKEN header with delete token
	req.Header.Set("X-LAST9-API-TOKEN", fmt.Sprintf("Bearer %s", deleteToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// NotificationChannelAttachRequest represents the request body for attaching a channel to an entity
type NotificationChannelAttachRequest struct {
	EntityID string `json:"entity_id"`
	Severity string `json:"severity"`
}

// NotificationAttachment represents an attachment record (child channel linked to entity)
type NotificationAttachment struct {
	ID        int    `json:"id"`
	ChannelID int    `json:"channel_id"`
	EntityID  string `json:"entity_id"`
	Severity  string `json:"severity"`
}

// AttachNotificationChannel attaches a notification channel to an entity with a severity level
func (c *Client) AttachNotificationChannel(channelID int, req *NotificationChannelAttachRequest) (*NotificationDestination, error) {
	var result NotificationDestination
	err := c.Post(fmt.Sprintf("/notification_settings/%d/attach", channelID), req, &result)
	return &result, err
}

// DetachNotificationChannel detaches a notification channel from an entity
// The API uses DELETE with entity_id as a query parameter
func (c *Client) DetachNotificationChannel(channelID int, entityID string) error {
	return c.Delete(fmt.Sprintf("/notification_settings/%d/attach?entity_id=%s", channelID, entityID))
}

// KPI methods
func (c *Client) CreateKPI(entityID string, req *KPICreateRequest) (*KPI, error) {
	var result KPI
	err := c.Post(fmt.Sprintf("/entities/%s/kpis", entityID), req, &result)
	return &result, err
}

func (c *Client) GetKPI(entityID, kpiID string) (*KPI, error) {
	var result KPI
	err := c.Get(fmt.Sprintf("/entities/%s/kpis/%s", entityID, kpiID), &result)
	return &result, err
}

func (c *Client) UpdateKPI(entityID, kpiID string, req *KPIUpdateRequest) (*KPI, error) {
	var result KPI
	err := c.Put(fmt.Sprintf("/entities/%s/kpis/%s", entityID, kpiID), req, &result)
	return &result, err
}

func (c *Client) DeleteKPI(entityID, kpiID string) error {
	return c.Delete(fmt.Sprintf("/entities/%s/kpis/%s", entityID, kpiID))
}
