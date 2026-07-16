# Databricks CLI v0.295+ — Full Command Reference

## Command Groups

The Databricks CLI organizes commands into functional groups. This reference covers the groups most relevant to data engineering workflows.

### Developer Tools

| Command | Purpose |
|---------|---------|
| `databricks bundle` | Databricks Asset Bundle lifecycle (deploy, run, destroy) |
| `databricks sync` | Synchronize a local directory to a workspace directory |

### Databricks Workspace

| Command | Purpose |
|---------|---------|
| `databricks fs` | Filesystem operations on DBFS and UC Volumes (ls, cp, cat, rm, mkdir) |
| `databricks workspace` | List, import, export, and delete notebooks and folders |
| `databricks repos` | Manage git repos in the workspace |
| `databricks secrets` | Manage secrets, secret scopes, and access permissions |

### Compute

| Command | Purpose |
|---------|---------|
| `databricks clusters` | Create, start, edit, list, terminate, and delete clusters |
| `databricks cluster-policies` | Control cluster configuration rules |
| `databricks libraries` | Install and uninstall libraries on clusters |
| `databricks instance-pools` | Manage pre-allocated cloud instances for faster cluster start |

### Lakeflow (Jobs & Pipelines)

| Command | Purpose |
|---------|---------|
| `databricks jobs` | Create, edit, delete, run, and monitor jobs |
| `databricks pipelines` | Manage Delta Live Tables / Spark Declarative Pipelines |

### Databricks SQL

| Command | Purpose |
|---------|---------|
| `databricks warehouses` | Manage SQL warehouses (create, start, stop, list) |
| `databricks queries` | CRUD operations on saved SQL queries |
| `databricks query-history` | Retrieve past query execution history |
| `databricks alerts` | Manage SQL alerts |

### Unity Catalog

| Command | Purpose |
|---------|---------|
| `databricks catalogs` | First layer of namespace (list, get, create, delete) |
| `databricks schemas` | Second layer — organizes tables, views, functions |
| `databricks tables` | Third layer — list, get, create, check existence |
| `databricks volumes` | File storage within Unity Catalog |
| `databricks functions` | User-Defined Functions (UDFs) |
| `databricks grants` | Manage access permissions on UC objects |
| `databricks metastores` | Top-level container of Unity Catalog objects |
| `databricks connections` | External data source connections |
| `databricks external-locations` | Cloud storage paths with credentials |
| `databricks storage-credentials` | Authentication for cloud storage |

### Identity and Access Management

| Command | Purpose |
|---------|---------|
| `databricks current-user` | Info about the authenticated user/service principal |
| `databricks groups` | Manage identity groups |
| `databricks permissions` | Read, write, update access on objects |
| `databricks service-principals` | Manage service principals |
| `databricks users` | Manage user identities |

### Machine Learning

| Command | Purpose |
|---------|---------|
| `databricks experiments` | MLflow experiment management |
| `databricks model-registry` | Workspace Model Registry |
| `databricks registered-models` | Unity Catalog Model Registry |
| `databricks serving-endpoints` | Model serving endpoints |

### Authentication

| Command | Purpose |
|---------|---------|
| `databricks auth login` | Interactive browser-based login |
| `databricks auth describe` | Show current auth credentials and source |
| `databricks auth profiles` | List profiles from ~/.databrickscfg |
| `databricks auth switch` | Set the default profile |
| `databricks auth token` | Get authentication token |

### Utility

| Command | Purpose |
|---------|---------|
| `databricks api` | Perform raw Databricks API calls |
| `databricks configure` | Configure authentication interactively |
| `databricks labs` | Manage Databricks Labs installations |

## Global Flags

These flags work with any command:

| Flag | Purpose |
|------|---------|
| `--debug` | Enable debug logging |
| `-o, --output type` | Output format: `text` (default) or `json` |
| `-p, --profile string` | Use a specific `~/.databrickscfg` profile |
| `-t, --target string` | Bundle target to use (if applicable) |

## SQL Statement Execution API

The `databricks api` command enables direct SQL queries against warehouses. This is the primary way to inspect Delta table data from the CLI.

### Endpoint

```
POST /api/2.0/sql/statements
```

### Request Body

```json
{
  "warehouse_id": "<id>",
  "statement": "<SQL>",
  "wait_timeout": "30s",
  "disposition": "INLINE",
  "format": "JSON_ARRAY"
}
```

### Useful SQL Statements for Delta Tables

```sql
-- Table structure
DESCRIBE TABLE EXTENDED catalog.schema.table

-- Delta version history
DESCRIBE HISTORY catalog.schema.table

-- Table properties (Delta metadata)
SHOW TBLPROPERTIES catalog.schema.table

-- Column statistics
ANALYZE TABLE catalog.schema.table COMPUTE STATISTICS FOR ALL COLUMNS

-- Partitioning info
SHOW PARTITIONS catalog.schema.table

-- Sample data
SELECT * FROM catalog.schema.table LIMIT 10

-- Row count
SELECT COUNT(*) FROM catalog.schema.table

-- Column value distribution
SELECT column, COUNT(*) as cnt FROM catalog.schema.table GROUP BY column ORDER BY cnt DESC LIMIT 20

-- Null analysis
SELECT
  COUNT(*) as total_rows,
  COUNT(column) as non_null,
  COUNT(*) - COUNT(column) as null_count
FROM catalog.schema.table

-- Schema comparison between tables
DESCRIBE TABLE catalog.schema.table_a
DESCRIBE TABLE catalog.schema.table_b

-- Time travel (read previous version)
SELECT * FROM catalog.schema.table VERSION AS OF 5 LIMIT 10
SELECT * FROM catalog.schema.table TIMESTAMP AS OF '2026-01-01' LIMIT 10
```

## Stack Overflow Bundle Configuration Patterns

Based on the `structured-data-pipelines` repository at `~/code/structured-data-pipelines/`.

### Bundle Location

Bundles are stored in `data_pipeline_bundles/` with each pipeline as a separate bundle directory:

```
data_pipeline_bundles/
├── _braze_events_bundle/
├── _cloudflare_bundle/
├── _data_dump_bundle/
├── _fivetran_bundle/
├── _google_ad_manager_bundle/
├── _google_analytics_bundle/
├── _product_events_pipelines_bundle/
├── _se_network_bundle/
├── _stack_exchange_bundle/
├── _teams_metrics_events_pipelines_bundle/
└── _unioned_ingest_bundle/
```

### Common Bundle Variables

Variables used across bundles:

| Variable | Purpose |
|----------|---------|
| `warehouse_id` | SQL warehouse for SQL tasks |
| `client` | Client/catalog identifier (e.g., `public_platform`) |
| `product` | Product/schema identifier (e.g., `product_events`) |
| `environment` | Deployment environment (dev/prod) |
| `data_eng_sec_group` | Data engineering security group for permissions |
| `data_eng_sp` | Service principal for job execution |
| `databricks_host` | Workspace URL |
| `job_run_schedule_utc` | Cron-like schedule |
| `github_actor` | GitHub user who triggered deployment (dev mode) |
| `notification_webhook_id` | Slack notification webhook |
| `databricks_uc_catalog` | Unity Catalog target catalog |
| `databricks_uc_schema` | Unity Catalog target schema |

### GitHub Actions Integration

Bundles are deployed via GitHub Actions workflows. User-specific workflows (postfixed `_user`) deploy in development mode using `github.actor` for isolation.

Key environment variables for CI:
- `DATABRICKS_HOST` — workspace URL
- `DATABRICKS_TOKEN` — authentication token
- `BUNDLE_VAR_*` — bundle variables passed as env vars (e.g., `BUNDLE_VAR_client`, `BUNDLE_VAR_environment`)

### Data Lake Zones

The project organizes data into zones:

1. **Raw Zone** — Direct ingestion from source systems using Delta Live Tables Autoloader
2. **Structured Zone** — Business logic transformations
3. **Logging Zone** — Pipeline execution logs and monitoring
4. **Metadata Zone** — Configuration tables and pipeline metadata
