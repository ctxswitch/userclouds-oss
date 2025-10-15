# ucctl - UserClouds CLI Tool

`ucctl` is a command-line utility for interacting with UserClouds installations. It provides tools for managing contexts, creating users, and syncing tenant resources across environments.

## Installation

Build the tool from source:

```bash
go build -o ucctl ./cmd/ucctl
```

Install to your PATH:

```bash
go install ./cmd/ucctl
```

## Commands

### Context Management

The context commands manage UserClouds environment configurations, similar to `kubectl` contexts in Kubernetes. Contexts store connection information (URL, client ID, and client secret) for different UserClouds installations.

#### `ucctl context` (alias: `ctx`)

Manage UserClouds contexts for different environments.

**Subcommands:**

##### `ucctl context list`

List all configured contexts. Shows which context is currently active with a `*` indicator.

```bash
ucctl context list
# or
ucctl ctx list
```

**Example output:**
```
CURRENT   NAME    URL                           CLIENT-ID
*         local   http://localhost:8080         test-client
          prod    https://prod.userclouds.com   prod-client
```

##### `ucctl context set <context-name>`

Create or update a context configuration.

**Flags:**
- `--url` (required) - UserClouds base URL
- `--client-id` (required) - OAuth2 client ID
- `--client-secret` (required) - OAuth2 client secret

```bash
ucctl context set local \
  --url http://localhost:8080 \
  --client-id my-client \
  --client-secret my-secret

ucctl ctx set prod \
  --url https://prod.userclouds.com \
  --client-id prod-client \
  --client-secret prod-secret
```

**Notes:**
- If this is the first context, it will automatically become the current context
- Context configurations are stored in `~/.userclouds/config.yaml`
- Client secrets are stored in plain text in the config file

##### `ucctl context use <context-name>`

Switch to a different context.

```bash
ucctl context use prod
# or
ucctl ctx use local
```

##### `ucctl context show`

Display the currently active context with connection details.

```bash
ucctl context show
```

**Example output:**
```
Current context: prod
URL: https://prod.userclouds.com
Client ID: prod-client
Client Secret: ****cret
```

**Note:** The client secret is masked, showing only the last 4 characters.

##### `ucctl context delete <context-name>`

Delete a context configuration.

```bash
ucctl context delete staging
```

**Note:** If you delete the current context, no context will be active.

---

### Create Commands

Commands for creating UserClouds resources.

#### `ucctl create user`

Create a new user with password or OIDC authentication.

**Connection Flags (provide either via flags or use context):**
- `--url` - IDP URL (or use current context)
- `--client-id` - OAuth2 client ID (or use current context)
- `--client-secret` - OAuth2 client secret (or use current context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)
- `--use-context` - Explicitly use the current context from config

**User Flags:**
- `--email` - User email address
- `--name` - User display name
- `--organization-id` - Organization ID for the user

**Password Authentication Flags:**
- `--username` - Username for password authentication
- `--password` - Password for password authentication

**OIDC Authentication Flags:**
- `--oidc-provider` - OIDC provider (e.g., `google`, `github`)
- `--oidc-issuer-url` - OIDC issuer URL
- `--oidc-subject` - OIDC subject ID

**Other Flags:**
- `-v, --verbose` - Enable verbose logging

**Examples:**

Create user with password authentication using current context:
```bash
ucctl create user \
  --username john.doe \
  --password SecurePass123! \
  --email john.doe@example.com \
  --name "John Doe"
```

Create user with password authentication using explicit flags:
```bash
ucctl create user \
  --url http://localhost:8080 \
  --client-id my-client \
  --client-secret my-secret \
  --username jane.smith \
  --password SecurePass456! \
  --email jane.smith@example.com \
  --name "Jane Smith"
```

Create user with OIDC authentication:
```bash
ucctl create user \
  --oidc-provider google \
  --oidc-issuer-url https://accounts.google.com \
  --oidc-subject 1234567890 \
  --email user@example.com \
  --name "OIDC User"
```

Create user with organization:
```bash
ucctl create user \
  --username admin \
  --password AdminPass123! \
  --email admin@example.com \
  --organization-id 550e8400-e29b-41d4-a716-446655440000
```

**Notes:**
- You must provide either username/password OR OIDC authentication details, not both
- If no connection flags are provided, the command will use the current context
- Connection flags override context settings when specified
- Client secret can be provided via flag, environment variable, or context

---

### Sync Commands

Commands for synchronizing resources between environments.

#### `ucctl sync tenant`

Sync authorization resources from a source tenant to a destination tenant. This command supports both context-based configuration and explicit URL/credential flags.

**Source Flags:**
- `--source` - Source context name (alternative to explicit URL/credentials)
- `--source-url` - Source tenant URL (required if not using --source)
- `--source-client-id` - Source OAuth2 client ID (required if not using --source)
- `--source-client-secret-var` - Environment variable containing source client secret (default: `UC_CLIENT_SECRET`)

**Destination Flags:**
- `--destination` - Destination context name (alternative to explicit URL/credentials)
- `--destination-url` - Destination tenant URL (required if not using --destination)
- `--destination-client-id` - Destination OAuth2 client ID (required if not using --destination)
- `--destination-client-secret-var` - Environment variable containing destination client secret (default: `UC_CLIENT_SECRET`)

**Sync Options:**
- `--dry-run` - Preview changes without applying them
- `--insert-only` - Only insert new resources, don't delete existing ones
- `-v, --verbose` - Enable verbose logging

**Examples:**

Sync using contexts (recommended):
```bash
# Set up contexts once
ucctl ctx set staging --url https://staging.userclouds.com --client-id staging-client --client-secret staging-secret
ucctl ctx set prod --url https://prod.userclouds.com --client-id prod-client --client-secret prod-secret

# Sync from staging to prod (dry run)
ucctl sync tenant --source staging --destination prod --dry-run

# Apply the sync
ucctl sync tenant --source staging --destination prod
```

Sync using explicit URLs and credentials:
```bash
export UC_CLIENT_SECRET=staging-secret
export UC_DEST_SECRET=prod-secret

ucctl sync tenant \
  --source-url https://staging.userclouds.com \
  --source-client-id staging-client \
  --destination-url https://prod.userclouds.com \
  --destination-client-id prod-client \
  --destination-client-secret-var UC_DEST_SECRET \
  --dry-run
```

Insert-only mode (no deletions):
```bash
ucctl sync tenant \
  --source staging \
  --destination prod \
  --insert-only
```

Verbose output with dry run:
```bash
ucctl sync tenant \
  --source staging \
  --destination prod \
  --dry-run \
  --verbose
```

**Sync Behavior:**
1. Fetches all AuthZ resources from source tenant (object types, objects, edge types, edges)
2. Fetches all AuthZ resources from destination tenant
3. Computes differences between source and destination
4. **Deletion phase** (unless `--insert-only`):
   - Deletes resources in destination that don't exist in source
   - Deletes in dependency order: Edges → EdgeTypes → Objects → ObjectTypes
5. **Insertion phase**:
   - Inserts resources from source that don't exist in destination
   - Inserts in dependency order: ObjectTypes → Objects → EdgeTypes → Edges
6. Skips all modifications if `--dry-run` is specified

**Notes:**
- Currently only supports AuthZ (authorization) resources
- You can mix context and explicit flags (e.g., use context for source, explicit URL for destination)
- Context credentials can be overridden by setting the environment variable
- Use `--dry-run` first to preview changes before applying
- Use `--insert-only` when you only want to add new resources without removing any
- The sync is one-way from source to destination
- Resources are compared by ID; matching IDs with different content will be updated

---

## Configuration

### Config File Location

Context configurations are stored in:
```
~/.userclouds/config.yaml
```

### Config File Format

```yaml
current_context: local
contexts:
  local:
    url: http://localhost:8080
    client_id: test-client
    client_secret: test-secret
  prod:
    url: https://prod.userclouds.com
    client_id: prod-client
    client_secret: prod-secret
```

**Security Note:** The config file contains client secrets in plain text. Ensure the file has appropriate permissions (mode 0600).

---

## Environment Variables

- `UC_CLIENT_SECRET` - Default environment variable for client secrets (used by `create user` and `sync tenant` commands)
- Custom environment variables can be specified via `--client-secret-var` and `--source-client-secret-var`/`--destination-client-secret-var` flags

---

## Common Workflows

### Setting Up Multiple Environments

```bash
# Configure local development environment
ucctl ctx set local \
  --url http://localhost:8080 \
  --client-id local-client \
  --client-secret local-secret

# Configure staging environment
ucctl ctx set staging \
  --url https://staging.userclouds.com \
  --client-id staging-client \
  --client-secret staging-secret

# Configure production environment
ucctl ctx set prod \
  --url https://prod.userclouds.com \
  --client-id prod-client \
  --client-secret prod-secret

# List all contexts
ucctl ctx list

# Switch to staging
ucctl ctx use staging
```

### Creating Users Across Environments

```bash
# Create user in local environment
ucctl ctx use local
ucctl create user --username testuser --password Test123! --email test@example.com

# Create same user in staging
ucctl ctx use staging
ucctl create user --username testuser --password Test123! --email test@example.com
```

### Syncing Configuration Between Environments

```bash
# Using contexts (recommended)
ucctl ctx set staging --url https://staging.userclouds.com --client-id staging-client --client-secret staging-secret
ucctl ctx set local --url https://local.userclouds.com --client-id local-client --client-secret local-secret

# Dry run: preview what would change
ucctl sync tenant --source staging --destination local --dry-run --verbose

# Apply the sync
ucctl sync tenant --source staging --destination local

# Or using explicit URLs (without contexts)
export UC_CLIENT_SECRET=staging-secret
export UC_DEST_SECRET=local-secret
ucctl sync tenant \
  --source-url https://staging.userclouds.com \
  --source-client-id staging-client \
  --destination-url https://local.userclouds.com \
  --destination-client-id local-client \
  --destination-client-secret-var UC_DEST_SECRET \
  --dry-run
```

---

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
ucctl completion bash > /etc/bash_completion.d/ucctl

# Zsh
ucctl completion zsh > "${fpath[1]}/_ucctl"

# Fish
ucctl completion fish > ~/.config/fish/completions/ucctl.fish

# PowerShell
ucctl completion powershell > ucctl.ps1
```

---

## Troubleshooting

### "No current context set" error

This occurs when no context is configured or active. Fix it by:

```bash
# Set and use a context
ucctl ctx set myenv --url <url> --client-id <id> --client-secret <secret>

# Or explicitly provide connection flags
ucctl create user --url <url> --client-id <id> --client-secret <secret> ...
```

### "Client secret is not set" error

This occurs when using environment variable-based secrets. Fix it by:

```bash
# Set the environment variable
export UC_CLIENT_SECRET=your-secret-here

# Or use the --client-secret flag directly
ucctl create user --client-secret your-secret ...
```

### Context config file permission issues

If you encounter permission errors:

```bash
# Fix config file permissions
chmod 600 ~/.userclouds/config.yaml
chmod 700 ~/.userclouds
```

---

## Development

### Building

```bash
go build -o ucctl ./cmd/ucctl
```

### Testing

```bash
go test ./cmd/ucctl/...
```

---

## Future Enhancements

- Support for more resource types in `synctenant` (tokenizer, userstore, authn, logserver)
- Encrypted storage for client secrets in config file
- Support for token refresh and session management
- Additional user management commands (update, delete, list)
- Organization management commands
- Batch operations support

---

## License

See the root repository LICENSE file for licensing information.
