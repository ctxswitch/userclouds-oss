# UserClouds Command Line Tools Documentation

## `cmd/ucconfig` - Configuration as Code Tool

**ucconfig** is a **configuration-as-code tool** for UserClouds tenants that uses **Infrastructure as Code (IaC)** principles, similar to how Terraform works for cloud infrastructure.

### Purpose
It allows you to manage UserClouds tenant resources (IDP/userstore resources like columns, accessors, mutators, purposes, etc.) as declarative configuration files (manifests) rather than managing them manually through the API or UI.

### Two Main Commands

#### 1. `ucconfig gen-manifest` - Export Configuration
- **What it does**: Fetches all live resources from a tenant and generates a manifest file
- **Output**: Creates a JSON or YAML manifest file describing all resources
- **Use case**: Export your current tenant configuration to version control or for backup
- **Additional feature**: Creates an external values directory to store attribute values separately

#### 2. `ucconfig apply` - Apply Configuration
- **What it does**: Applies a manifest file to modify a tenant to match the desired state
- **How it works**:
  1. Reads the manifest (JSON or YAML)
  2. Fetches current live resources from the tenant
  3. **Generates Terraform configuration** internally
  4. **Generates Terraform state** for existing resources
  5. Runs `terraform init` and `terraform apply` to reconcile differences
- **Flags**:
  - `--dry-run`: Preview changes without applying (runs `terraform plan`)
  - `--auto-approve`: Skip confirmation prompt
  - `--tf-provider-version-constraint`: Specify terraform provider version
  - `--tf-provider-dev-dir-path`: Use local dev build of terraform provider

### Architecture

**Manifest Structure:**
```json
{
  "resources": [
    {
      "uc_terraform_type": "userstore_column",
      "manifest_id": "userstore_column_email",
      "resource_uuids": {
        "__DEFAULT": "uuid-here",
        "tenant-name": "uuid-here"
      },
      "attributes": {
        "name": "email",
        "type": "string",
        "index_type": "indexed"
      }
    }
  ]
}
```

**Key Features:**
- **Multi-tenant support**: Resources can have different UUIDs per tenant using `resource_uuids` mapping
- **Terraform-backed**: Leverages the official `terraform-provider-userclouds` under the hood
- **Declarative**: Describe what you want, not how to get there
- **State management**: Automatically generates Terraform state for existing resources to enable proper diff/reconciliation

### Use Cases

1. **Version Control**: Store tenant configurations in git for change tracking and code review
2. **Environment Promotion**: Export config from dev → apply to staging/prod
3. **Disaster Recovery**: Backup tenant configurations and restore if needed
4. **Reproducible Deployments**: Define infrastructure as code for consistent deployments
5. **Multi-tenant Management**: Maintain a single manifest that works across multiple tenant environments

### Example Workflow

```bash
# Export current tenant config
ucconfig gen-manifest my-tenant-config.yaml

# Edit the manifest file to add/modify resources
vim my-tenant-config.yaml

# Preview changes
ucconfig apply my-tenant-config.yaml --dry-run

# Apply changes to tenant
ucconfig apply my-tenant-config.yaml
```

This tool essentially provides **GitOps-style configuration management** for UserClouds tenants, making tenant management more programmatic, repeatable, and auditable.

---

## `cmd/runaccessors` - Accessor Testing Tool

### Purpose

The `runaccessors` command is a testing and load testing tool designed to execute UserClouds accessors against a tenant's data store. Accessors are data access patterns that define what user data can be retrieved and how it's filtered, controlled by access policies. This tool allows developers and operators to test accessor functionality, verify data retrieval, and perform load testing by running accessors multiple times with different search parameters.

### Functionality

The tool executes a specified accessor multiple times (configurable via iterations) against a UserClouds tenant using provided search strings. For each iteration, it runs the accessor with each search string provided as selector values, retrieves the resulting data, and logs the output. The tool authenticates with the tenant using stored credentials, executes the accessor via the UserClouds IDP (Identity Provider) API endpoint, and displays the returned data fields. Results are logged with timing information, and the tool can optionally continue on errors or stop at the first failure. All operations are timed to help assess accessor performance under different loads.

### Command-Line Options

- `--tenant` (required): Tenant ID or name against which the accessor will be executed
- `--accessor-id` (required): UUID of the accessor to run
- `search` (required, positional): One or more search strings to use as selector values for the accessor
- `--iterations`: Number of times to run the complete test (default: 1)
- `--verbose`: Enable verbose debug output to console
- `--logfile`: Path to a log file for detailed debug output (optional)
- `--continue-on-error`: Continue executing subsequent searches if an error occurs instead of stopping immediately

### Use Cases

This tool is primarily used for testing and validating accessor configurations in UserClouds environments. Common use cases include verifying that newly created accessors return the expected data fields for specific search criteria, load testing accessors to measure performance and response times under repeated execution, debugging accessor behavior by examining detailed logs of requests and responses, and validating that accessor access policies work correctly across different tenant configurations. It's particularly useful during development and QA phases when iterating on accessor definitions and access control policies.

---

## `cmd/azcli` - Authorization Graph Management Tool

### Primary Purpose

`azcli` is a command-line interface for managing a graph-based authorization system. It provides direct access to UserClouds' authorization (authz) service, enabling administrators and developers to create, query, and manipulate the authorization graph that models permissions and relationships between objects. The tool authenticates using OAuth2 client credentials (client ID and secret) against a specified tenant URL.

### Key Functionality

The tool operates on a **relationship-based access control (ReBAC)** model with four core primitives:

1. **Object Types**: Define categories of entities (e.g., users, documents, groups)
2. **Objects**: Concrete instances of object types with optional aliases
3. **Edge Types**: Define typed relationships between object types with attributes (direct, inherit, or propagate permissions)
4. **Edges**: Concrete relationship instances connecting two objects

The authorization system supports sophisticated permission modeling through three attribute types: **direct** (source has attribute on target), **inherit** (source inherits attributes the target has), and **propagate** (attributes on source propagate to target). The tool also includes attribute checking capabilities to verify permission paths between objects, organization management for multi-tenancy, and migration commands for moving objects/edge types between organizations.

### Command-Line Interface

The tool uses a positional argument structure:
```
azcli <tenant URL> <client ID> <client secret> <command> [command-specific args]
```

**Core Commands:**
- Object Type Operations: `listobjtypes`, `createobjecttype <name>`
- Edge Type Operations: `listedgetypes`, `createedgetype <name> <src-type-id> <tgt-type-id> <org-id> <attributes>`
- Object Operations: `listobjs`, `createobj <type-id> <alias> <org-id>`, `deleteobj <obj-id>`
- Edge Operations: `listedges <obj-id>`, `createedge <type-id> <src-id> <tgt-id>`, `deleteedge <edge-id>`
- Organization Operations: `listorgs`, `createorg <name> <region>`
- Permission Queries: `checkattr <src-obj-id> <tgt-obj-id> <attribute>`, `listattr <src-obj-id> <tgt-obj-id>`
- Migration: `migrateobj <obj-id> <org-id>`, `migrateedgetype <et-id> <org-id>`
- Special: `sendinvite` for delegation invitations (plex client functionality)

### Use Cases

This tool is designed for:
- **Development & Testing**: Quickly scaffold authorization graphs, create test scenarios, and verify permission logic
- **Operations & Administration**: Manage multi-tenant authorization configurations, migrate resources between organizations
- **Debugging**: Inspect the authorization graph structure, trace permission paths, and validate attribute relationships
- **Bulk Operations**: Script creation of authorization schemas and perform administrative tasks programmatically

The tool is particularly valuable for systems implementing fine-grained, relationship-based access control where permissions are derived from graph traversal rather than simple role assignments.

---

## `cmd/dataimport` - Bulk Data Import Tool

### Primary Purpose

The `dataimport` command is a standalone utility for importing user data into the UserClouds Identity Provider (IDP) system. It processes specially formatted data files containing user records and imports them into a specific tenant's user store using the UserClouds mutator framework. The tool is designed to handle large-scale data imports from local files or S3 buckets.

### Key Functionality

The tool reads data files with a specific binary format that includes:
- A 37-byte header containing the mutator ID (UUID)
- A client context record (access policy context) delimited by `\x02`
- Multiple data records, each containing user selector values and row data separated by `\x01` delimiter

Each record is parsed and executed through the UserClouds mutator engine, which applies the appropriate access policies and stores the data in the tenant's user store. The import process includes progress tracking, error handling, and validation. It tolerates up to 100 bad records before failing, logging warnings for invalid records while continuing to process valid ones. Status updates are logged every 100 records to track progress.

### Command-Line Interface

The tool requires two positional arguments:
- `<tenantID>`: UUID of the target tenant where data will be imported
- `<filePath>`: Absolute or relative path to the data import file

Environment variables must be set:
- `UC_UNIVERSE`: The UserClouds universe to connect to
- `UC_REGION`: The UserClouds region to use

There are no optional flags or switches; the tool uses a simple positional argument interface.

### Use Cases

The `dataimport` command is primarily used for:
- **Bulk user data migration**: Importing large datasets of user records into UserClouds from external systems
- **Initial tenant setup**: Populating a new tenant with existing user data during onboarding
- **Data restoration**: Restoring user data from backups or archived exports
- **Automated import pipelines**: Can be integrated with S3-based workflows where the worker system monitors S3 buckets for uploaded data files, automatically processes them, tracks job status, and deletes successfully imported files

---

## `cmd/migrate` - Database Migration Management Tool

### Primary Purpose

The `migrate` tool is a database migration management utility for the UserClouds platform. It handles schema migrations for multiple database types across different environments (dev, staging, production), supporting both single-service databases and multi-tenant architectures. The tool manages bidirectional migrations (up and down) with built-in safety checks to prevent data loss and schema inconsistencies.

### Key Functionality

The migrate command operates on a universe-based configuration system (determined by environment variables) and performs schema migrations by:

- Comparing the current database schema version against available migrations in code
- Executing SQL migrations sequentially from the current version to a requested version
- Storing migration history in a dedicated migrations table for rollback capability
- Validating that existing database migrations match the codebase before applying new changes
- Supporting special handling for multi-tenant databases, where it iterates through all tenants and applies migrations to each tenant's database and associated region databases
- Providing a "check deployed" mode to verify that deployed environments are synchronized with local migration code

The tool includes safety mechanisms such as preventing unsafe operations in production, prompting for confirmation before downgrades, detecting migration mismatches between code and database, and validating migration sequences for correctness.

### Main Command-Line Flags

- `--code`: Use down migrations from code instead of database (dev environments only)
- `--checkDeployed`: Verify that the specified universe's migrations are up-to-date with local code
- `--logfile <path>`: Specify a logfile for debug output
- `--noDowngradePrompt`: Skip prompts when database version equals code version (useful for automated setups)
- `--noPrompt`: Disable all user prompts in non-production environments (implies --noDowngradePrompt)
- `--noUnsafeWarnInDev`: Suppress warnings about unsafe migrations in development
- `--verbose`: Enable verbose output for detailed logging

### Use Cases

The migrate tool is designed for several key scenarios:

1. **Development Environment Setup**: Use `migrate --noPrompt <database>` during local development setup to automatically bring databases to the latest version without manual intervention
2. **Production Deployments**: Run `migrate --checkDeployed <database>` in CI/CD pipelines to ensure deployed services match expected migration versions before releasing
3. **Multi-Tenant Migrations**: Execute `migrate tenantdb` to apply schema changes across all tenant databases, including their primary databases and region-specific replicas
4. **Schema Rollbacks**: Downgrade to a specific version when reverting problematic changes, with built-in safety checks to prevent data loss
5. **Cross-Branch Development**: Handle migration conflicts when switching between feature branches that have divergent schema changes

The tool supports multiple database types including `tenantdb`, `companyconfig`, `logdb`, and `status`, each with their own migration definitions and baseline versions.

---

## `cmd/provision` - Infrastructure Provisioning Tool

### Primary Purpose

The `provision` command is a system administration tool for managing the lifecycle of UserClouds infrastructure resources. It handles the creation, validation, updating, and deletion of companies, tenants, and events within the UserClouds platform. The tool operates directly against databases and configuration storage, supporting both interactive confirmation workflows and batch operations across multiple resources.

### Key Functionality

The tool supports six main operations on three resource types (company, tenant, events):

- **provision**: Creates or updates resources based on JSON files or database records, with interactive prompts for data conflicts and overwrites
- **validate**: Verifies resources are correctly configured without making changes
- **deprovision**: Soft-deletes resources with user confirmation
- **nuke**: Hard-deletes underlying database resources (tenant only, dev environment only)
- **setowner**: Assigns a user as admin/owner of a company and propagates ownership to all associated tenants
- **genfile**: Generates a JSON provisioning file from an existing resource for replication or backup

Resources can be targeted by UUID, JSON filename, or the keyword "all" for batch operations. The tool includes built-in safety checks like production confirmations, diff comparisons between existing and new configurations, and automatic rollback prompts on errors. For tenants, it supports parallel provisioning with a batch provisioner for improved performance.

### Command-Line Flags

- `--simulate`: Validate target objects without mutating database state (dry-run mode)
- `--verbose`: Enable verbose logging output
- `--owner <uuid>`: UUID of user to mark as admin/owner (company/setowner operation only)
- `--logfile <path>`: Specify logfile name for debug output
- `--deep`: Enable deep provisioning that validates and corrects relationships between system objects
- `--useBaselineSchema`: Force migration-by-migration provisioning from baseline schema instead of using final schema create statements

### Use Cases

**Initial Infrastructure Setup**: Bootstrap new companies and tenants from JSON configuration files during platform deployment or customer onboarding.

**Configuration Management**: Update existing tenant or company configurations across environments, with built-in diff checking to prevent unintended changes.

**Validation & Auditing**: Run `validate` operations to verify infrastructure consistency without making changes, useful for compliance checks or before major updates.

**Disaster Recovery**: Generate JSON snapshots of existing resources with `genfile` for backup purposes, or reprovision resources from these files to restore services or replicate configurations to new environments.

---
