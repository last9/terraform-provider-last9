# API Call Sequences for Terraform Provider Resources

This document details the sequence of API calls made for CRUD operations on Alert Groups (Entities) and Alerts.

**Base URL**: `{api_base_url}/api/v4/organizations/{org}`

---

## Alert Groups (Entities)

Alert Groups are represented as `last9_entity` resources in Terraform.

### Create

**Terraform operation**: `terraform apply` (new resource)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Create Entity                                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /entities                                                               │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "name": "my-service",                                                      │
│   "type": "service",                                                         │
│   "external_ref": "my-service-ref",                                          │
│   "description": "My service description",                                   │
│   "data_source": "levitate",                                                 │
│   "namespace": "production",                                                 │
│   "tier": "critical",                                                        │
│   "workspace": "default",                                                    │
│   "entity_class": "alert-manager",                                           │
│   "ui_readonly": true,                                                       │
│   "indicators": [...],                                                       │
│   "notification_channels": [...]                                             │
│ }                                                                            │
│                                                                              │
│ Response: Entity object with ID                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Set Entity Metadata (if metadata fields specified)                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ PUT /entities/{entity_id}/metadata                                           │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "team": "platform",                                                        │
│   "tags": ["production", "critical"],                                        │
│   "labels": {"env": "prod", "region": "us-west"},                            │
│   "links": [{"name": "Dashboard", "url": "https://..."}],                    │
│   "adhoc_filter": {                                                          │
│     "data_source": "levitate",                                               │
│     "labels": {"service": "my-service"}                                      │
│   }                                                                          │
│ }                                                                            │
│                                                                              │
│ Note: POST /entities does NOT accept metadata fields (tags, labels, team,    │
│       links, adhoc_filter). These must be set via separate PUT call.         │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: Read Entity (to sync state)                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /entities/{entity_id}                                                    │
│                                                                              │
│ Response: Full entity object including nested metadata                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Read

**Terraform operation**: `terraform plan`, `terraform refresh`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ GET /entities/{entity_id}                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ Response:                                                                    │
│ {                                                                            │
│   "id": "entity-uuid",                                                       │
│   "name": "my-service",                                                      │
│   "type": "service",                                                         │
│   "external_ref": "my-service-ref",                                          │
│   "description": "...",                                                      │
│   "data_source": "levitate",                                                 │
│   "data_source_id": "ds-uuid",                                               │
│   "namespace": "production",                                                 │
│   "tier": "critical",                                                        │
│   "workspace": "default",                                                    │
│   "entity_class": "alert-manager",                                           │
│   "ui_readonly": true,                                                       │
│   "indicators": [...],                                                       │
│   "notification_channels": [...],                                            │
│   "metadata": {                            ◄── Metadata is nested            │
│     "team": "platform",                                                      │
│     "tags": ["production", "critical"],                                      │
│     "labels": {"env": "prod"},                                               │
│     "links": [...],                                                          │
│     "adhoc_filter": {...}                                                    │
│   }                                                                          │
│ }                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Update

**Terraform operation**: `terraform apply` (existing resource changed)

The update is split into two API calls depending on which fields changed:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ If CORE fields changed (name, type, description, namespace, tier, etc.)      │
├─────────────────────────────────────────────────────────────────────────────┤
│ PUT /entities/{entity_id}                                                    │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "name": "updated-service",                                                 │
│   "type": "service",                                                         │
│   "data_source_id": "ds-uuid",           ◄── Mandatory for PUT               │
│   "description": "Updated description",                                      │
│   "namespace": "staging",                                                    │
│   "tier": "high",                                                            │
│   "workspace": "default",                                                    │
│   "ui_readonly": false                                                       │
│ }                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ If METADATA fields changed (tags, labels, team, links, adhoc_filter)         │
├─────────────────────────────────────────────────────────────────────────────┤
│ PUT /entities/{entity_id}/metadata                                           │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "team": "infrastructure",              ◄── Team is required                │
│   "tags": ["staging", "test"],                                               │
│   "labels": {"env": "staging"},                                              │
│   "links": [...],                                                            │
│   "adhoc_filter": {...}                                                      │
│ }                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Read Entity (to sync state)                                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /entities/{entity_id}                                                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Delete

**Terraform operation**: `terraform destroy`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ DELETE /entities/{entity_id}                                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│ Note: Requires delete-scoped token (delete_token or delete_refresh_token)    │
│                                                                              │
│ Response: 200 OK on success                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Import

**Terraform operation**: `terraform import last9_entity.name <entity_id>`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ GET /entities/{entity_id}                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ Import ID format: entity_id                                                  │
│ Example: terraform import last9_entity.my_service abc123-uuid                │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Alerts

Alerts are represented as `last9_alert` resources in Terraform. Alerts require a parent Entity and automatically manage an associated KPI.

### Create

**Terraform operation**: `terraform apply` (new resource)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Create KPI for the Alert                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /entities/{entity_id}/kpis                                              │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "name": "my-alert-a1b2c3d4",           ◄── Generated: {alert_name}-{token} │
│   "definition": {                                                            │
│     "query": "sum(rate(http_requests_total[5m]))",                           │
│     "source": "levitate",                                                    │
│     "unit": "count"                                                          │
│   },                                                                         │
│   "kpi_type": "custom"                                                       │
│ }                                                                            │
│                                                                              │
│ Response: KPI object with ID                                                 │
│ Note: KPI is auto-created and managed by the provider                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Create Alert Rule                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /entities/{entity_id}/alert-rules                                       │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "rule_name": "High Error Rate",                                            │
│   "primary_indicator": "my-alert-a1b2c3d4",   ◄── KPI name from Step 1       │
│   "severity": "breach",                                                      │
│   "is_disabled": false,                                                      │
│   "group_timeseries_notifications": true,                                    │
│   "condition": "expr > 100.000000",                                          │
│   "alert_condition": "count_true(result) >= 5",                              │
│   "eval_window": 10,                                                         │
│   "expression_args": {                                                       │
│     "my-alert-a1b2c3d4": {                ◄── KPI name                       │
│       "id": "kpi-uuid"                    ◄── KPI ID from Step 1             │
│     }                                                                        │
│   },                                                                         │
│   "properties": {                                                            │
│     "description": "Alert when error rate exceeds threshold",                │
│     "runbook": {"link": "https://runbook.example.com/..."},                  │
│     "annotations": {"team": "platform"}                                      │
│   },                                                                         │
│   "notification_channels": ["channel-id-1", "channel-id-2"]                  │
│ }                                                                            │
│                                                                              │
│ Response: Alert object with ID                                               │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: Read Alert (to sync state)                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /entities/{entity_id}/alert-rules/{alert_id}                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ Rollback: If Alert creation fails                                            │
├─────────────────────────────────────────────────────────────────────────────┤
│ DELETE /entities/{entity_id}/kpis/{kpi_id}                                   │
│                                                                              │
│ Note: KPI is cleaned up if alert creation fails                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Read

**Terraform operation**: `terraform plan`, `terraform refresh`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ GET /entities/{entity_id}/alert-rules/{alert_id}                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ Response:                                                                    │
│ {                                                                            │
│   "id": "alert-uuid",                                                        │
│   "rule_name": "High Error Rate",                                            │
│   "description": "...",                                                      │
│   "entity_id": "entity-uuid",                                                │
│   "primary_indicator": "my-alert-a1b2c3d4",                                  │
│   "expression": "my-alert-a1b2c3d4",      ◄── Computed from KPI name         │
│   "condition": "expr > 100.000000",                                          │
│   "eval_window": 10,                                                         │
│   "alert_condition": "count_true(result) >= 5",                              │
│   "severity": "breach",                                                      │
│   "mute_until": 0,                                                           │
│   "is_disabled": false,                                                      │
│   "properties": {...},                                                       │
│   "group_timeseries_notifications": true,                                    │
│   "notification_channels": [...]                                             │
│ }                                                                            │
│                                                                              │
│ Note: `is_disabled` is NOT read from API response to avoid drift.            │
│       The value from Terraform config is preserved.                          │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Update

**Terraform operation**: `terraform apply` (existing resource changed)

If the alert name or query changes, a new KPI is created first:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Create new KPI (if name or query changed)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /entities/{entity_id}/kpis                                              │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "name": "updated-alert-e5f6g7h8",      ◄── New generated name              │
│   "definition": {                                                            │
│     "query": "sum(rate(http_errors_total[5m]))",   ◄── Updated query         │
│     "source": "levitate",                                                    │
│     "unit": "count"                                                          │
│   },                                                                         │
│   "kpi_type": "custom"                                                       │
│ }                                                                            │
│                                                                              │
│ Response: New KPI object with ID                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Update Alert Rule                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ PUT /entities/{entity_id}/alert-rules/{alert_id}                             │
│                                                                              │
│ Request Body (ALL fields required - no partial updates):                     │
│ {                                                                            │
│   "rule_name": "Updated Alert Name",                                         │
│   "primary_indicator": "updated-alert-e5f6g7h8",                             │
│   "severity": "threat",                                                      │
│   "is_disabled": false,                                                      │
│   "group_timeseries_notifications": true,                                    │
│   "condition": "expr > 200.000000",                                          │
│   "alert_condition": "count_true(result) >= 3",                              │
│   "eval_window": 5,                                                          │
│   "expression_args": {                                                       │
│     "updated-alert-e5f6g7h8": {                                              │
│       "id": "new-kpi-uuid"                                                   │
│     }                                                                        │
│   },                                                                         │
│   "properties": {...},                                                       │
│   "notification_channels": [...]                                             │
│ }                                                                            │
│                                                                              │
│ IMPORTANT: PUT deletes and recreates the alert. Response contains NEW ID.    │
│                                                                              │
│ Response: New Alert object (may have different ID!)                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: Delete old KPI (if new KPI was created)                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ DELETE /entities/{entity_id}/kpis/{old_kpi_id}                               │
│                                                                              │
│ Note: Errors are ignored - KPI may already be gone                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 4: Read Alert (to sync state with new ID)                               │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /entities/{entity_id}/alert-rules/{new_alert_id}                         │
│                                                                              │
│ Note: Resource ID in Terraform state is updated to the new alert ID          │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ Rollback: If Alert update fails (and new KPI was created)                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ DELETE /entities/{entity_id}/kpis/{new_kpi_id}                               │
│                                                                              │
│ Note: New KPI is cleaned up if alert update fails                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Delete

**Terraform operation**: `terraform destroy`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Delete Alert Rule                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ DELETE /entities/{entity_id}/alert-rules/{alert_id}                          │
│                                                                              │
│ Note: Requires delete-scoped token (delete_token or delete_refresh_token)    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Delete associated KPI                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ DELETE /entities/{entity_id}/kpis/{kpi_id}                                   │
│                                                                              │
│ Note: Errors are ignored - KPI may already be gone                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Import

**Terraform operation**: `terraform import last9_alert.name <entity_id>:<alert_id>`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Import uses composite ID format: entity_id:alert_id                          │
├─────────────────────────────────────────────────────────────────────────────┤
│ Example: terraform import last9_alert.my_alert abc123:def456                 │
│                                                                              │
│ The provider parses the composite ID and:                                    │
│ 1. Sets entity_id = "abc123"                                                 │
│ 2. Sets resource ID = "def456"                                               │
│ 3. Calls GET /entities/abc123/alert-rules/def456                             │
│                                                                              │
│ Note: Some computed fields (query, kpi_id, kpi_name, is_disabled) may not    │
│       match exactly after import since they are computed during create.      │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Authentication

All API calls require authentication via the `X-LAST9-API-TOKEN` header:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Token Refresh (if using refresh_token)                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST {api_base_url}/api/v4/oauth/access_token                                │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "refresh_token": "eyJ..."                                                  │
│ }                                                                            │
│                                                                              │
│ Response:                                                                    │
│ {                                                                            │
│   "access_token": "eyJ...",                                                  │
│   "expires_at": 1737500000,                                                  │
│   "issued_at": 1737400000,                                                   │
│   "type": "bearer",                                                          │
│   "scopes": ["write"]                                                        │
│ }                                                                            │
│                                                                              │
│ Note: Token is cached and refreshed 5 minutes before expiry                  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│ Request Header Format                                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ X-LAST9-API-TOKEN: Bearer {access_token}                                     │
│                                                                              │
│ For DELETE operations:                                                       │
│ - Uses delete_refresh_token to get delete-scoped access token, OR            │
│ - Uses static delete_token directly (legacy mode)                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Summary Table

| Resource | Operation | API Calls |
|----------|-----------|-----------|
| Entity | Create | POST /entities → PUT /entities/{id}/metadata → GET /entities/{id} |
| Entity | Read | GET /entities/{id} |
| Entity | Update (core) | PUT /entities/{id} → GET /entities/{id} |
| Entity | Update (metadata) | PUT /entities/{id}/metadata → GET /entities/{id} |
| Entity | Delete | DELETE /entities/{id} |
| Alert | Create | POST /kpis → POST /alert-rules → GET /alert-rules/{id} |
| Alert | Read | GET /alert-rules/{id} |
| Alert | Update | POST /kpis (if needed) → PUT /alert-rules/{id} → DELETE /kpis (old) → GET /alert-rules/{new_id} |
| Alert | Delete | DELETE /alert-rules/{id} → DELETE /kpis/{id} |
