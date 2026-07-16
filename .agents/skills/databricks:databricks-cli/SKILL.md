---
name: databricks-cli
description: Guide for using the Databricks CLI with Databricks Asset Bundles (DAB) for deploying pipelines, inspecting Delta tables, managing Unity Catalog objects, running jobs, and exploring data. This skill should be used when working with Databricks workspaces, deploying bundles, querying catalog/schema/table metadata, inspecting Delta tables, managing clusters, running or monitoring jobs, syncing files, or authenticating with Databricks. Triggers on phrases like "deploy to Databricks", "run a bundle", "inspect Delta table", "list tables", "Databricks job", "check cluster", "databricks auth", "bundle validate", or any Databricks CLI operation.
---

# Databricks CLI — Asset Bundles & Data Engineering

The Databricks CLI (v0.200+) provides a unified interface for managing Databricks workspaces, deploying Databricks Asset Bundles (DAB), and inspecting data. This skill covers the modern DAB-based workflow — not the legacy `dbx` tool.

## Prerequisites

- **Databricks CLI** installed (`databricks --version` to verify, requires >= 0.200)
- **Authentication** configured via `~/.databrickscfg` or environment variables
- **Azure Databricks** workspace access (Stack Overflow uses Azure-hosted Databricks)

## Authentication

To authenticate for the first time or refresh credentials:

```bash
# Interactive browser-based login (recommended for local dev)
databricks auth login --host https://<workspace-url>

# Check current auth status
databricks auth describe

# List configured profiles
databricks auth profiles

# Switch default profile
databricks auth switch --profile <profile-name>

# Use a specific profile for any command
databricks <command> --profile <profile-name>
```

Authentication methods in order of preference:
1. **OAuth (browser login)** — `databricks auth login` — recommended for interactive use
2. **Personal Access Token (PAT)** — set in `~/.databrickscfg` or `DATABRICKS_TOKEN` env var
3. **Environment variables** — `DATABRICKS_HOST` + `DATABRICKS_TOKEN`

## Databricks Asset Bundles (DAB)

DAB is the modern deployment model. A bundle is defined by a `databricks.yml` file that declares jobs, pipelines, variables, targets, and permissions.

### Core Bundle Commands

```bash
# Validate bundle configuration (catch errors before deploying)
databricks bundle validate

# Preview what would change without deploying
databricks bundle plan --target dev

# Deploy to a target environment
databricks bundle deploy --target dev
databricks bundle deploy --target prod

# Run a specific job or pipeline defined in the bundle
databricks bundle run <job_name>

# View deployed resources
databricks bundle summary

# Open a resource in the browser
databricks bundle open <resource_name>

# Destroy deployed resources (cleanup dev environments)
databricks bundle destroy --target dev

# Pass bundle variables via CLI
databricks bundle deploy --target dev --var="client=public_platform" --var="environment=dev"
```

### Bundle Targets

Bundles support multiple targets defined in `databricks.yml`:

| Target | Mode | Purpose |
|--------|------|---------|
| `user` | `development` | Personal dev sandbox, isolated per developer |
| `dev` | `production` | Shared dev environment (triggers paused by default) |
| `prod` | `production` | Production deployment |

Development mode (`mode: development`) automatically prefixes resources with `[dev]` and the deployer identity to prevent collisions.

### Bundle File Sync

To sync local files to the workspace for iterative development:

```bash
databricks bundle sync --target dev
```

### Generating Bundle Config from Existing Resources

To import an existing job or pipeline into a bundle:

```bash
databricks bundle generate job --existing-job-id 123 --key my_job
databricks bundle generate pipeline --existing-pipeline-id abc --key my_pipeline
```

## Inspecting Data — Unity Catalog

Unity Catalog organizes data in a three-level namespace: **Catalog > Schema > Table**.

### Browse the Catalog Hierarchy

```bash
# List all catalogs
databricks catalogs list

# Get details about a specific catalog
databricks catalogs get <catalog_name>

# List schemas within a catalog
databricks schemas list <catalog_name>

# Get schema details
databricks schemas get <catalog_name>.<schema_name>

# List tables within a schema
databricks tables list <catalog_name> <schema_name>

# Get table metadata (columns, type, storage, properties)
databricks tables get <catalog_name>.<schema_name>.<table_name>

# Check if a table exists
databricks tables exists <catalog_name>.<schema_name>.<table_name>
```

### Inspecting Delta Tables

To inspect Delta table contents and structure, use the SQL Statement Execution API via a SQL warehouse. To find the warehouse ID, run `databricks warehouses list` and use the `id` field.

```bash
# Preview rows
databricks api post /api/2.0/sql/statements \
  --json '{
    "warehouse_id": "<warehouse_id>",
    "statement": "SELECT * FROM catalog.schema.table LIMIT 10",
    "wait_timeout": "30s"
  }'

# Describe table structure (columns, types, partitioning)
databricks api post /api/2.0/sql/statements \
  --json '{
    "warehouse_id": "<warehouse_id>",
    "statement": "DESCRIBE EXTENDED catalog.schema.table",
    "wait_timeout": "30s"
  }'

# Delta table version history (operations, timestamps, user)
databricks api post /api/2.0/sql/statements \
  --json '{
    "warehouse_id": "<warehouse_id>",
    "statement": "DESCRIBE HISTORY catalog.schema.table",
    "wait_timeout": "30s"
  }'

# Table properties (Delta-specific metadata)
databricks api post /api/2.0/sql/statements \
  --json '{
    "warehouse_id": "<warehouse_id>",
    "statement": "SHOW TBLPROPERTIES catalog.schema.table",
    "wait_timeout": "30s"
  }'

# Row count
databricks api post /api/2.0/sql/statements \
  --json '{
    "warehouse_id": "<warehouse_id>",
    "statement": "SELECT COUNT(*) FROM catalog.schema.table",
    "wait_timeout": "30s"
  }'

# Column-level statistics
databricks api post /api/2.0/sql/statements \
  --json '{
    "warehouse_id": "<warehouse_id>",
    "statement": "ANALYZE TABLE catalog.schema.table COMPUTE STATISTICS FOR ALL COLUMNS",
    "wait_timeout": "30s"
  }'
```

### Volumes (File Storage in Unity Catalog)

```bash
# List volumes in a schema
databricks volumes list <catalog_name> <schema_name>

# Read files from DBFS or UC Volumes
databricks fs ls dbfs:/path/to/directory
databricks fs ls /Volumes/<catalog>/<schema>/<volume>/

# Download a file
databricks fs cp dbfs:/path/to/file ./local-file

# Upload a file
databricks fs cp ./local-file dbfs:/path/to/destination
```

## Jobs & Pipelines

### Managing Jobs

```bash
# List all jobs
databricks jobs list

# Get details for a specific job
databricks jobs get <job_id>

# Trigger a job run
databricks jobs run-now <job_id>

# List recent runs for a job
databricks jobs list-runs --job-id <job_id>

# Get run output/results
databricks jobs get-run <run_id>
databricks jobs get-run-output <run_id>

# Cancel a running job
databricks jobs cancel-run <run_id>

# Cancel all runs of a job
databricks jobs cancel-all-runs <job_id>
```

### Managing Pipelines (Delta Live Tables / Lakeflow)

```bash
# List all pipelines
databricks pipelines list-pipelines

# Get pipeline details
databricks pipelines get <pipeline_id>

# Start a pipeline update
databricks pipelines run <pipeline_id>

# Stop a running pipeline
databricks pipelines stop <pipeline_id>

# View pipeline event history
databricks pipelines list-pipeline-events <pipeline_id>

# View update history
databricks pipelines list-updates <pipeline_id>
```

## Clusters

```bash
# List all clusters
databricks clusters list

# Get cluster details
databricks clusters get <cluster_id>

# Start a terminated cluster
databricks clusters start <cluster_id>

# Terminate a cluster
databricks clusters delete <cluster_id>

# List available Spark versions
databricks clusters spark-versions

# List available node types
databricks clusters list-node-types
```

## Secrets Management

```bash
# List secret scopes
databricks secrets list-scopes

# List secrets in a scope (values are redacted)
databricks secrets list-secrets <scope_name>

# Create a secret
databricks secrets put-secret <scope_name> <key_name> --string-value "value"

# Delete a secret
databricks secrets delete-secret <scope_name> <key_name>
```

## Workspace Files

```bash
# List workspace contents
databricks workspace list /path/to/directory

# Export a notebook
databricks workspace export /path/to/notebook --format SOURCE

# Import a notebook
databricks workspace import ./local-notebook.py /path/to/destination
```

## Output Formatting

All commands support `--output json` for machine-readable output:

```bash
databricks tables list my_catalog my_schema --output json
databricks jobs list --output json | jq '.[] | {job_id, settings.name}'
```

## Common Workflows

### Deploy and Run a Pipeline End-to-End

```bash
databricks bundle validate
databricks bundle deploy --target dev
databricks bundle run <job_name>
databricks jobs list-runs --job-id <job_id>  # monitor progress
```

### Explore an Unfamiliar Catalog

```bash
databricks catalogs list
databricks schemas list <catalog>
databricks tables list <catalog> <schema>
databricks tables get <catalog>.<schema>.<table>
```

### Quick Data Preview

```bash
WAREHOUSE=$(databricks warehouses list --output json | jq -r '.[0].id')
databricks api post /api/2.0/sql/statements \
  --json "{\"warehouse_id\": \"$WAREHOUSE\", \"statement\": \"SELECT * FROM catalog.schema.table LIMIT 5\", \"wait_timeout\": \"30s\"}"
```

## Reference

For the full CLI command tree, all subcommand groups, and project-specific bundle configuration patterns, see `references/cli-reference.md`.
