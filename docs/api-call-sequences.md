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

## Drop Rules

Drop Rules are represented as `last9_drop_rule` resources in Terraform. They use a **list-based CRUD pattern** where the entire list of rules is replaced on each operation.

**Important**: The API manages rules as a list. Each create/update/delete operation:
1. GETs the current list of all rules
2. Modifies the list (add/update/remove)
3. POSTs the entire modified list back

### Create

**Terraform operation**: `terraform apply` (new resource)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Get existing rules                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/routing?region={region}                                   │
│                                                                              │
│ Response:                                                                    │
│ {                                                                            │
│   "properties": [                                                            │
│     { "name": "existing-rule-1", ... },                                      │
│     { "name": "existing-rule-2", ... }                                       │
│   ]                                                                          │
│ }                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Check for duplicate name                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ If rule with same name exists → Error: "drop rule X already exists"          │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: POST updated list with new rule appended                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /logs_settings/routing?region={region}&cluster_id={cluster_id}          │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "properties": [                                                            │
│     { "name": "existing-rule-1", ... },     ◄── Existing rules               │
│     { "name": "existing-rule-2", ... },                                      │
│     {                                        ◄── New rule appended           │
│       "name": "my-drop-rule",                                                │
│       "telemetry": "logs",                                                   │
│       "filters": [                                                           │
│         {                                                                    │
│           "key": "attributes[\"service\"]",                                  │
│           "value": "debug-service",                                          │
│           "operator": "equals",                                              │
│           "conjunction": "and"              ◄── Optional, for multiple       │
│         }                                                                    │
│       ],                                                                     │
│       "action": {                                                            │
│         "name": "drop",                                                      │
│         "destination": "/dev/null",                                          │
│         "properties": {}                                                     │
│       }                                                                      │
│     }                                                                        │
│   ]                                                                          │
│ }                                                                            │
│                                                                              │
│ Note: cluster_id is MANDATORY for POST operations                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 4: Read rule (via list) to sync state                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/routing?region={region}                                   │
│                                                                              │
│ Find rule by name in response                                                │
│                                                                              │
│ Terraform ID format: {region}:{cluster_id}:{rule_name}                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Read

**Terraform operation**: `terraform plan`, `terraform refresh`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ GET /logs_settings/routing?region={region}                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ Response:                                                                    │
│ {                                                                            │
│   "properties": [                                                            │
│     {                                                                        │
│       "name": "my-drop-rule",                                                │
│       "telemetry": "logs",                                                   │
│       "filters": [...],                                                      │
│       "action": {...}                                                        │
│     },                                                                       │
│     ...                                                                      │
│   ]                                                                          │
│ }                                                                            │
│                                                                              │
│ Provider finds the rule by name from the list                                │
│ If not found → resource is removed from state                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Update

**Terraform operation**: `terraform apply` (existing resource changed)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Get existing rules                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/routing?region={region}                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Replace matching rule in list                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ Find rule by name → Replace with updated rule                                │
│ If not found → Error: "drop rule X not found for update"                     │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: POST updated list                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /logs_settings/routing?region={region}&cluster_id={cluster_id}          │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "properties": [                                                            │
│     { "name": "other-rule", ... },                                           │
│     { "name": "my-drop-rule", ... }         ◄── Updated rule                 │
│   ]                                                                          │
│ }                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 4: Read rule to sync state                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/routing?region={region}                                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Delete

**Terraform operation**: `terraform destroy`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Get existing rules                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/routing?region={region}                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Remove rule from list                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ Filter out rule by name                                                      │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: POST list without the deleted rule                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /logs_settings/routing?region={region}&cluster_id={cluster_id}          │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "properties": [                                                            │
│     { "name": "remaining-rule-1", ... },                                     │
│     { "name": "remaining-rule-2", ... }                                      │
│   ]                                                  ◄── Deleted rule absent │
│ }                                                                            │
│                                                                              │
│ Note: Requires delete-scoped token                                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Import

**Terraform operation**: `terraform import last9_drop_rule.name <region>:<cluster_id>:<rule_name>`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Import ID format: region:cluster_id:rule_name                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ Example: terraform import last9_drop_rule.my_rule ap-south-1:abc-123:my-rule │
│                                                                              │
│ The provider:                                                                │
│ 1. Parses the composite ID                                                   │
│ 2. Calls GET /logs_settings/routing?region=ap-south-1                        │
│ 3. Finds rule by name in response                                            │
│ 4. Sets region, cluster_id, and rule attributes in state                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Forward Rules

Forward Rules are represented as `last9_forward_rule` resources in Terraform. Like Drop Rules, they use a **list-based CRUD pattern**.

### Create

**Terraform operation**: `terraform apply` (new resource)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Get existing rules                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/forward?region={region}                                   │
│                                                                              │
│ Response:                                                                    │
│ {                                                                            │
│   "properties": [...]                                                        │
│ }                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Check for duplicate name                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ If rule with same name exists → Error                                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: POST updated list with new rule appended                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /logs_settings/forward?region={region}&cluster_id={cluster_id}          │
│                                                                              │
│ Request Body:                                                                │
│ {                                                                            │
│   "properties": [                                                            │
│     ...existing rules...,                                                    │
│     {                                                                        │
│       "name": "my-forward-rule",                                             │
│       "telemetry": "logs",                                                   │
│       "destination": "https://logs.example.com/ingest",                      │
│       "filters": [                                                           │
│         {                                                                    │
│           "key": "attributes[\"service\"]",                                  │
│           "value": "important-service",                                      │
│           "operator": "equals"                                               │
│         }                                                                    │
│       ]                                                                      │
│     }                                                                        │
│   ]                                                                          │
│ }                                                                            │
│                                                                              │
│ Note: cluster_id is MANDATORY for POST operations                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 4: Read rule to sync state                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/forward?region={region}                                   │
│                                                                              │
│ Terraform ID format: {region}:{cluster_id}:{rule_name}                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Read

**Terraform operation**: `terraform plan`, `terraform refresh`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ GET /logs_settings/forward?region={region}                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ Response:                                                                    │
│ {                                                                            │
│   "properties": [                                                            │
│     {                                                                        │
│       "name": "my-forward-rule",                                             │
│       "telemetry": "logs",                                                   │
│       "destination": "https://logs.example.com/ingest",                      │
│       "filters": [...]                                                       │
│     },                                                                       │
│     ...                                                                      │
│   ]                                                                          │
│ }                                                                            │
│                                                                              │
│ Provider finds the rule by name from the list                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Update

**Terraform operation**: `terraform apply` (existing resource changed)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: GET existing rules                                                   │
│ Step 2: Replace matching rule by name                                        │
│ Step 3: POST updated list                                                    │
│ Step 4: Read to sync state                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ Same pattern as Drop Rules - see above                                       │
│                                                                              │
│ POST /logs_settings/forward?region={region}&cluster_id={cluster_id}          │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Delete

**Terraform operation**: `terraform destroy`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: GET existing rules                                                   │
│ Step 2: Filter out rule by name                                              │
│ Step 3: POST list without deleted rule                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ Same pattern as Drop Rules - see above                                       │
│                                                                              │
│ POST /logs_settings/forward?region={region}&cluster_id={cluster_id}          │
│                                                                              │
│ Note: Requires delete-scoped token                                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Import

**Terraform operation**: `terraform import last9_forward_rule.name <region>:<cluster_id>:<rule_name>`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Import ID format: region:cluster_id:rule_name                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ Example: terraform import last9_forward_rule.my_rule ap-south-1:abc:my-rule  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Scheduled Search Alerts

Scheduled Search Alerts are represented as `last9_scheduled_search_alert` resources in Terraform. Unlike Drop/Forward Rules, they use **individual REST CRUD operations** (not list-based).

### Create

**Terraform operation**: `terraform apply` (new resource)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Fetch notification destination details                               │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /notification_settings                                                   │
│                                                                              │
│ Response: Array of notification destinations                                 │
│ [                                                                            │
│   { "id": 1295, "name": "slack-alerts", "type": "slack", ... },              │
│   { "id": 1296, "name": "pagerduty", "type": "pagerduty", ... }              │
│ ]                                                                            │
│                                                                              │
│ Provider finds destination by ID from user config                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Create scheduled search alert                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ POST /logs_settings/scheduled_search?region={region}                         │
│                                                                              │
│ Request Body (single alert object, NOT array):                               │
│ {                                                                            │
│   "rule_name": "High Error Rate Alert",                                      │
│   "rule_type": "scheduled_search",                                           │
│   "query_type": "logjson-aggregate",                                         │
│   "physical_index": "logs",                                                  │
│   "properties": {                                                            │
│     "telemetry": "logs",                                                     │
│     "query": "[{\"type\":\"filter\",\"query\":{...}}]",                      │
│     "post_processor": [                                                      │
│       {                                                                      │
│         "type": "aggregate",                                                 │
│         "aggregates": [                                                      │
│           { "function": {"$count": []}, "as": "error_count" }                │
│         ],                                                                   │
│         "groupby": {}                                                        │
│       }                                                                      │
│     ],                                                                       │
│     "search_frequency": 300,                  ◄── Seconds (5 minutes)        │
│     "threshold": {                                                           │
│       "operator": ">",                                                       │
│       "value": 100                                                           │
│     },                                                                       │
│     "alert_destinations": [                   ◄── Full destination objects   │
│       { "id": 1295, "name": "slack-alerts", ... }                            │
│     ]                                                                        │
│   }                                                                          │
│ }                                                                            │
│                                                                              │
│ Response: Created alert with ID                                              │
│ {                                                                            │
│   "id": "alert-uuid-123",                                                    │
│   "rule_name": "High Error Rate Alert",                                      │
│   ...                                                                        │
│ }                                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: Read alert to sync state                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/scheduled_search?region={region}                          │
│                                                                              │
│ Find alert by name in response array                                         │
│                                                                              │
│ Terraform ID format: {region}:{alert_id}:{rule_name}                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Read

**Terraform operation**: `terraform plan`, `terraform refresh`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ GET /logs_settings/scheduled_search?region={region}                          │
├─────────────────────────────────────────────────────────────────────────────┤
│ Response: Array of scheduled search alerts                                   │
│ [                                                                            │
│   {                                                                          │
│     "id": "alert-uuid-123",                                                  │
│     "rule_name": "High Error Rate Alert",                                    │
│     "rule_type": "scheduled_search",                                         │
│     "query_type": "logjson-aggregate",                                       │
│     "physical_index": "logs",                                                │
│     "region": "ap-south-1",                                                  │
│     "properties": {                                                          │
│       "telemetry": "logs",                                                   │
│       "query": "...",                                                        │
│       "post_processor": [...],                                               │
│       "search_frequency": 300,                                               │
│       "threshold": { "operator": ">", "value": 100 },                        │
│       "alert_destinations": [...]           ◄── Contains internal IDs        │
│     },                                                                       │
│     "created_at": 1737500000,                                                │
│     "updated_at": 1737500000                                                 │
│   },                                                                         │
│   ...                                                                        │
│ ]                                                                            │
│                                                                              │
│ Provider finds alert by name in array                                        │
│                                                                              │
│ Note: alert_destinations is NOT read from API response because               │
│       the API returns internal association IDs, not the notification         │
│       destination IDs the user specified. User's config is preserved.        │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Update

**Terraform operation**: `terraform apply` (existing resource changed)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 1: Fetch notification destinations (if destinations changed)            │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /notification_settings                                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 2: Update scheduled search alert                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ PUT /logs_settings/scheduled_search/{alert_id}?region={region}               │
│                                                                              │
│ Request Body (single alert object):                                          │
│ {                                                                            │
│   "rule_name": "Updated Alert Name",                                         │
│   "rule_type": "scheduled_search",                                           │
│   "query_type": "logjson-aggregate",                                         │
│   "physical_index": "logs",                                                  │
│   "properties": {                                                            │
│     "telemetry": "logs",                                                     │
│     "query": "...",                                                          │
│     "post_processor": [...],                                                 │
│     "search_frequency": 600,                  ◄── Updated                    │
│     "threshold": { "operator": ">", "value": 200 },  ◄── Updated             │
│     "alert_destinations": [...]                                              │
│   }                                                                          │
│ }                                                                            │
│                                                                              │
│ Response: Updated alert (may have new ID)                                    │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Step 3: Read alert to sync state                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ GET /logs_settings/scheduled_search?region={region}                          │
│                                                                              │
│ Update Terraform ID if alert ID changed                                      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Delete

**Terraform operation**: `terraform destroy`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ DELETE /logs_settings/scheduled_search/{alert_id}?region={region}            │
├─────────────────────────────────────────────────────────────────────────────┤
│ Note: Requires delete-scoped token                                           │
│                                                                              │
│ Response: 200 OK on success                                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Import

**Terraform operation**: `terraform import last9_scheduled_search_alert.name <region>:<alert_id>:<rule_name>`

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Import ID format: region:alert_id:rule_name                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ Example: terraform import last9_scheduled_search_alert.my_alert \            │
│          ap-south-1:abc-uuid:My-Alert                                        │
│                                                                              │
│ The provider:                                                                │
│ 1. Parses the composite ID                                                   │
│ 2. Calls GET /logs_settings/scheduled_search?region=ap-south-1               │
│ 3. Finds alert by name in response array                                     │
│                                                                              │
│ Note: alert_destinations is ignored during import since the API returns      │
│       internal IDs that don't match user-specified notification IDs.         │
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
| Drop Rule | Create | GET /logs_settings/routing → POST /logs_settings/routing (list with new rule) |
| Drop Rule | Read | GET /logs_settings/routing (find by name) |
| Drop Rule | Update | GET /logs_settings/routing → POST /logs_settings/routing (list with updated rule) |
| Drop Rule | Delete | GET /logs_settings/routing → POST /logs_settings/routing (list without rule) |
| Forward Rule | Create | GET /logs_settings/forward → POST /logs_settings/forward (list with new rule) |
| Forward Rule | Read | GET /logs_settings/forward (find by name) |
| Forward Rule | Update | GET /logs_settings/forward → POST /logs_settings/forward (list with updated rule) |
| Forward Rule | Delete | GET /logs_settings/forward → POST /logs_settings/forward (list without rule) |
| Scheduled Search | Create | GET /notification_settings → POST /logs_settings/scheduled_search |
| Scheduled Search | Read | GET /logs_settings/scheduled_search (find by name) |
| Scheduled Search | Update | GET /notification_settings → PUT /logs_settings/scheduled_search/{id} |
| Scheduled Search | Delete | DELETE /logs_settings/scheduled_search/{id} |

### API Pattern Comparison

| Resource Type | CRUD Pattern | Notes |
|---------------|--------------|-------|
| Entity, Alert | Individual REST | Standard create/read/update/delete per resource |
| Drop Rule, Forward Rule | List-based | GET all → modify list → POST entire list |
| Scheduled Search | Individual REST | Standard REST operations (POST/PUT/DELETE) |

**List-based CRUD Limitation**: Drop Rules and Forward Rules use a list-based pattern where the entire list is replaced on each operation. This has inherent race conditions - concurrent Terraform runs could overwrite each other's changes.
