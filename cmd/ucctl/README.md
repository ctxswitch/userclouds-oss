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

UserClouds has two types of contexts:
- **Console Tenant Contexts**: For managing console employees and companies
- **Regular Tenant Contexts**: For managing tenant-specific users and resources

Regular tenant contexts reference a console tenant context by name, avoiding credential duplication when multiple tenants share the same console.

#### `ucctl context` (alias: `ctx`)

Manage UserClouds contexts for different environments.

**Subcommands:**

##### `ucctl context list`

List all configured contexts. Shows which context is currently active with a `*` indicator, plus the context type (console or tenant).

```bash
ucctl context list
# or
ucctl ctx list
```

**Example output:**
```
CURRENT   NAME            TYPE      URL
*         console-prod    console   https://console.tenant.example.com
          tenant-foo      tenant    https://foo.tenant.example.com
          tenant-bar      tenant    https://bar.tenant.example.com
```

##### `ucctl context set <context-name>`

Create or update a context configuration.

**Flags:**
- `--url` (required) - UserClouds base URL
- `--client-id` (required) - OAuth2 client ID
- `--client-secret` (required) - OAuth2 client secret
- `--console` - Console tenant context name (required for regular tenant contexts)
- `--console-tenant` - Mark this context as a console tenant

**Creating a Console Tenant Context:**

```bash
ucctl context set console-prod \
  --url https://console.tenant.example.com \
  --client-id console-client \
  --client-secret console-secret \
  --console-tenant
```

**Creating a Regular Tenant Context:**

```bash
ucctl context set tenant-foo \
  --url https://foo.tenant.example.com \
  --client-id foo-client \
  --client-secret foo-secret \
  --console console-prod

ucctl ctx set tenant-bar \
  --url https://bar.tenant.example.com \
  --client-id bar-client \
  --client-secret bar-secret \
  --console console-prod
```

**Notes:**
- If this is the first context, it will automatically become the current context
- Context configurations are stored in `~/.userclouds/config.yaml`
- Client secrets are stored in plain text in the config file
- Console tenant contexts cannot reference other consoles (`--console` and `--console-tenant` are mutually exclusive)
- Regular tenant contexts must reference a console tenant via `--console`
- Multiple tenant contexts can share the same console tenant, avoiding credential duplication

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

**Example output for a regular tenant context:**
```
Current context: tenant-foo
Type: Tenant
URL: https://foo.tenant.example.com
Client ID: foo-client
Client Secret: ****cret
Console Tenant: console-prod
  Console URL: https://console.tenant.example.com
  Console Client ID: console-client
```

**Example output for a console tenant context:**
```
Current context: console-prod
Type: Console Tenant
URL: https://console.tenant.example.com
Client ID: console-client
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

Create a new user with password authentication, OIDC authentication, or without authentication.

**Important:** The user is created in the tenant specified by the current context:
- **For console employee users**: Switch to a console tenant context first
- **For regular tenant users**: Switch to a regular tenant context first

When creating a user without authentication, the authentication will be automatically added when the user logs in for the first time via OIDC (based on email match).

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**User Flags:**
- `--email` - User email address
- `--name` - User display name
- `--organization-id` - Organization ID for the user (required)

**Password Authentication Flags:**
- `--username` - Username for password authentication
- `--password` - Password for password authentication

**OIDC Authentication Flags:**
- `--oidc-provider` - OIDC provider (e.g., `google`, `github`)
- `--oidc-issuer-url` - OIDC issuer URL
- `--oidc-subject` - OIDC subject ID

**Other Flags:**
- `-a, --admin` - Grant admin privileges to the user
- `-v, --verbose` - Enable verbose logging

**Examples:**

**Creating Console Employee Users:**

Console employees are users who manage companies and other tenants. They are created in the console tenant.

```bash
# Switch to console tenant context
ucctl ctx use console-prod

# Create console employee with password authentication
ucctl create user \
  --username john.doe \
  --password SecurePass123! \
  --email john.doe@example.com \
  --name "John Doe" \
  --organization-id <console-org-id>

# Create console employee with admin privileges
ucctl create user \
  --username admin \
  --password AdminPass123! \
  --email admin@example.com \
  --name "Admin User" \
  --organization-id <console-org-id> \
  --admin
```

**Creating Regular Tenant Users:**

Regular tenant users belong to specific tenants and have access to that tenant's resources.

```bash
# Switch to tenant context
ucctl ctx use tenant-foo

# Create tenant user with password authentication
ucctl create user \
  --username customer1 \
  --password CustomerPass123! \
  --email customer@example.com \
  --name "Customer User" \
  --organization-id <tenant-org-id>

# Create tenant user with OIDC authentication
ucctl create user \
  --oidc-provider google \
  --oidc-issuer-url https://accounts.google.com \
  --oidc-subject 1234567890 \
  --email user@example.com \
  --name "Google User" \
  --organization-id <tenant-org-id>

# Create tenant user without authentication (will authenticate via OIDC on first login)
ucctl create user \
  --email newuser@example.com \
  --name "New User" \
  --organization-id <tenant-org-id>
```

**Using Explicit Connection Flags:**

If you don't want to use contexts, you can specify connection details explicitly:

```bash
ucctl create user \
  --url https://console.tenant.example.com \
  --client-id my-client \
  --client-secret my-secret \
  --username jane.smith \
  --password SecurePass456! \
  --email jane.smith@example.com \
  --name "Jane Smith" \
  --organization-id <org-id>
```

**Notes:**
- You must provide either username/password OR OIDC authentication details, OR omit both to create without authentication
- If no connection flags are provided, the command will use the current context
- Connection flags override context settings when specified
- Client secret can be provided via flag, environment variable, or context
- The `--admin` flag will grant admin privileges on the tenant/company after user creation

---

### Get Commands

Commands for retrieving UserClouds resources.

#### `ucctl get objecttypes`

List all AuthZ object types with pagination and filtering support.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Pagination Flags:**
- `-l, --limit` - Maximum number of results to return per page (0 = terminal height, default for interactive mode)
- `-c, --cursor` - Pagination cursor for fetching the next page
- `--no-pager` - Disable interactive paging and show all results at once

**Filter Flags:**
- `-f, --filter` - Filter results using boolean expressions (see [Filter Query Language](#filter-query-language))
  - Supported keys: `id`, `type_name`, `created`, `updated`
  - Example: `--filter "type_name=user%&organization_id=550e8400"`
- `--raw-filter` - Raw filter query (see [Filter Query Language](#filter-query-language) for format)
  - Example: `--raw-filter "('type_name',LK,'user%')"`
  - Mutually exclusive with `--filter`

**Examples:**

List all object types with interactive paging:
```bash
ucctl ctx use tenant-foo
ucctl get objecttypes
```

List all object types without paging (print all at once):
```bash
ucctl get objecttypes --no-pager
```

Filter by exact type name:
```bash
ucctl get objecttypes --filter "type_name=user"
```

Filter using wildcards (prefix match):
```bash
ucctl get objecttypes --filter "type_name=user%"
```

Filter with AND operator:
```bash
ucctl get objecttypes --filter "type_name=user&organization_id=550e8400-e29b-41d4-a716-446655440000"
```

Filter with OR operator:
```bash
ucctl get objecttypes --filter "type_name=user|type_name=admin"
```

Filter with grouping:
```bash
ucctl get objecttypes --filter "(type_name=user|type_name=admin)&organization_id=550e8400-e29b-41d4-a716-446655440000"
```

Using raw filter format:
```bash
ucctl get objecttypes --raw-filter "('type_name',LK,'user%')"
```

Using raw filter with composite query:
```bash
ucctl get objecttypes --raw-filter "(('type_name',LK,'user%'),OR,('type_name',EQ,'admin'))"
```

Using explicit connection flags:
```bash
ucctl get objecttypes \
  --url https://foo.tenant.example.com \
  --client-id my-client \
  --client-secret my-secret \
  --filter "type_name=document"
```

Manual pagination with cursor:
```bash
# Get first page with limit
ucctl get objecttypes --limit 10 --no-pager

# Get next page using cursor from previous response
ucctl get objecttypes --cursor "id:550e8400-..." --limit 10 --no-pager
```

**Notes:**
- By default, uses interactive paging when output is to a terminal
- Interactive mode displays results page by page based on terminal height
- Use `--no-pager` to disable interactive mode and print all results
- See [Filter Query Language](#filter-query-language) for comprehensive filtering documentation
- Timestamp format for display: `2006-01-02 15:04:05`

#### `ucctl get objecttype <id>`

Get detailed information about a specific AuthZ object type by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Examples:**

```bash
ucctl ctx use tenant-foo
ucctl get objecttype 550e8400-e29b-41d4-a716-446655440000
```

#### `ucctl get objects`

List all AuthZ objects with pagination and filtering support.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Pagination Flags:**
- `-l, --limit` - Maximum number of results to return per page (0 = terminal height, default for interactive mode)
- `-c, --cursor` - Pagination cursor for fetching the next page
- `--no-pager` - Disable interactive paging and show all results at once

**Filter Flags:**
- `-f, --filter` - Filter results using boolean expressions (see [Filter Query Language](#filter-query-language))
  - Supported keys: `id`, `alias`, `organization_id`, `type_id`, `created`, `updated`
  - Example: `--filter "alias=user1&type_id=550e8400"`
- `--raw-filter` - Raw filter query (see [Filter Query Language](#filter-query-language) for format)
  - Mutually exclusive with `--filter`

**Examples:**

```bash
# List all objects with interactive paging
ucctl get objects
# Filter by exact alias
ucctl get objects --filter "alias=admin"

# Filter by alias prefix using wildcard
ucctl get objects --filter "alias=admin%"

# Filter with AND: type and organization
ucctl get objects --filter "type_id=550e8400-e29b-41d4-a716-446655440000&organization_id=660e8400-e29b-41d4-a716-446655440001"

# Filter with OR: multiple aliases
ucctl get objects --filter "alias=admin|alias=superuser"
```

#### `ucctl get object <id>`

Get detailed information about a specific AuthZ object by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Examples:**

```bash
ucctl get object 550e8400-e29b-41d4-a716-446655440000
```

#### `ucctl get edgetypes`

List all AuthZ edge types with pagination and filtering support.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Pagination Flags:**
- `-l, --limit` - Maximum number of results to return per page (0 = terminal height, default for interactive mode)
- `-c, --cursor` - Pagination cursor for fetching the next page
- `--no-pager` - Disable interactive paging and show all results at once

**Filter Flags:**
- `-f, --filter` - Filter results using boolean expressions (see [Filter Query Language](#filter-query-language))
  - Supported keys: `id`, `type_name`, `organization_id`, `source_object_type_id`, `target_object_type_id`, `created`, `updated`
  - Example: `--filter "type_name=member&organization_id=550e8400"`
- `--raw-filter` - Raw filter query (see [Filter Query Language](#filter-query-language) for format)
  - Mutually exclusive with `--filter`

**Examples:**

```bash
# List all edge types with interactive paging
ucctl get edgetypes
# Filter by exact type name
ucctl get edgetypes --filter "type_name=member"

# Filter by type name prefix using wildcard
ucctl get edgetypes --filter "type_name=member%"

# Filter with AND: type name and organization
ucctl get edgetypes --filter "type_name=member&organization_id=550e8400-e29b-41d4-a716-446655440000"

# Filter with OR: multiple type names
ucctl get edgetypes --filter "type_name=member|type_name=admin"
```

#### `ucctl get edgetype <id>`

Get detailed information about a specific AuthZ edge type by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Examples:**

```bash
ucctl get edgetype 550e8400-e29b-41d4-a716-446655440000
```

#### `ucctl get edges`

List all AuthZ edges with pagination and filtering support.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Pagination Flags:**
- `-l, --limit` - Maximum number of results to return per page (0 = terminal height, default for interactive mode)
- `-c, --cursor` - Pagination cursor for fetching the next page
- `--no-pager` - Disable interactive paging and show all results at once

**Filter Flags:**
- `-f, --filter` - Filter results using boolean expressions (see [Filter Query Language](#filter-query-language))
  - Supported keys: `id`, `source_object_id`, `target_object_id`, `created`, `updated`
  - Example: `--filter "source_object_id=550e8400&target_object_id=660e8400"`
- `--raw-filter` - Raw filter query (see [Filter Query Language](#filter-query-language) for format)
  - Mutually exclusive with `--filter`

**Examples:**

```bash
# List all edges with interactive paging
ucctl get edges
# Filter by source object
ucctl get edges --filter "source_object_id=550e8400-e29b-41d4-a716-446655440000"

# Filter with AND: source and target objects
ucctl get edges --filter "source_object_id=550e8400-e29b-41d4-a716-446655440000&target_object_id=660e8400-e29b-41d4-a716-446655440001"

# Filter with OR: multiple source objects
ucctl get edges --filter "source_object_id=550e8400-e29b-41d4-a716-446655440000|source_object_id=660e8400-e29b-41d4-a716-446655440002"
```

#### `ucctl get edge <id>`

Get detailed information about a specific AuthZ edge by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Examples:**

```bash
ucctl get edge 550e8400-e29b-41d4-a716-446655440000
```

---

### Filter Query Language

The `get` commands support two types of filtering: simplified boolean expressions (via `--filter`) and raw filter queries (via `--raw-filter`).

#### Simplified Filter Syntax (`--filter`)

The `--filter` flag provides a user-friendly syntax for common filtering needs:

**Operators:**
- `&` - AND operator (e.g., `type_name=user&id=123`)
- `|` - OR operator (e.g., `type_name=user|type_name=admin`)
- `()` - Grouping (e.g., `(type_name=user|type_name=admin)&organization_id=456`)

**Wildcards:**
- `%` - Matches any number of characters (SQL LIKE)
- Works with any string field (e.g., `type_name=user%` matches "user", "user_admin", etc.)

**Examples:**
```bash
# Simple filter
--filter "type_name=user"

# AND operation
--filter "type_name=user&organization_id=123"

# OR operation
--filter "type_name=user|type_name=admin"

# Grouping with wildcards
--filter "(type_name=user%|type_name=admin)&organization_id=456"
```

#### Raw Filter Query Format (`--raw-filter`)

The `--raw-filter` flag provides direct access to the full pagination filter query language. This format gives you complete control over filter operations, including access to additional operators not available in the simplified syntax.

**Query Types:**

1. **LEAF Query** - A single comparison:
   
```
   ('KEY',OPERATOR,'VALUE')
   
```

2. **NESTED Query** - A grouped query:
   
```
   (FILTER_QUERY)
   
```

3. **COMPOSITE Query** - Multiple queries combined with logical operators:
   
```
   (FILTER_QUERY,LOGICAL_OP,FILTER_QUERY)
   (FILTER_QUERY,LOGICAL_OP,FILTER_QUERY,LOGICAL_OP,FILTER_QUERY)
   
```

**Operators:**

| Type | Operator | SQL Equivalent | Description |
|------|----------|----------------|-------------|
| **COMPARISON** | `EQ` | `=` | Equal to |
| | `NE` | `!=` | Not equal to |
| | `GT` | `>` | Greater than |
| | `GE` | `>=` | Greater than or equal to |
| | `LT` | `<` | Less than |
| | `LE` | `<=` | Less than or equal to |
| **PATTERN** | `LK` | `LIKE` | Pattern match with `%` and `_` wildcards |
| | `NL` | `NOT LIKE` | Negated pattern match |
| **ARRAY** | `HAS` | `ANY` | Check if value exists in array |
| **LOGICAL** | `AND` | `AND` | Logical AND (higher precedence) |
| | `OR` | `OR` | Logical OR (lower precedence) |

**Key Types:**

Different field types support different operators:

- **StringKeyType**: Supports COMPARISON and PATTERN operators
- **IntKeyType**: Supports COMPARISON operators only
- **BoolKeyType**: Supports COMPARISON operators only
- **UUIDKeyType**: Supports COMPARISON operators only
- **TimestampKeyType**: Supports COMPARISON operators only (value in microseconds since epoch)
- **ArrayKeyType**: Supports ARRAY operators only
- **UUIDArrayKeyType**: Supports ARRAY operators only
- **Nullable types**: Can match NULL or their base type

**Pattern Matching:**

For PATTERN operators (`LK`, `NL`):
- `%` matches zero or more characters
- `_` matches exactly one character
- Escape special characters with `\` (e.g., `\%` matches literal `%`, `\_` matches literal `_`)

**Examples:**

```bash
# Simple equality
--raw-filter "('type_name',EQ,'user')"

# Pattern matching with wildcard
--raw-filter "('type_name',LK,'user%')"

# Greater than comparison
--raw-filter "('created',GT,'1609459200000000')"

# Composite AND query
--raw-filter "(('type_name',EQ,'user'),AND,('organization_id',EQ,'550e8400-e29b-41d4-a716-446655440000'))"

# Composite OR query
--raw-filter "(('type_name',LK,'user%'),OR,('type_name',EQ,'admin'))"

# Complex nested query with grouping
--raw-filter "((('type_name',LK,'user%'),OR,('type_name',EQ,'admin')),AND,('organization_id',EQ,'550e8400-e29b-41d4-a716-446655440000'))"

# Array membership check
--raw-filter "('tags',HAS,'production')"

# Negated pattern match
--raw-filter "('type_name',NL,'test%')"
```

**Operator Precedence:**

When multiple logical operators appear in a composite query without explicit grouping, standard SQL precedence applies:
- All consecutive `AND` operations are grouped together first
- Then `OR` operations are evaluated

For example: `(A,OR,B,AND,C,AND,D)` is evaluated as `(A,OR,(B,AND,C,AND,D))`

To control evaluation order, use nested queries: `((A,OR,B),AND,(C,OR,D))`

**Important Notes:**

- The `--filter` and `--raw-filter` flags are mutually exclusive - you cannot use both at once
- Field names (keys) must be valid for the resource type being queried
- String values with quotes must be escaped: use `\'` or `\"`
- The raw filter format is passed directly to the API without client-side validation

---

### Delete Commands

Commands for deleting UserClouds resources.

**Important:** Delete operations are permanent and cannot be undone. All delete commands require explicit confirmation before proceeding, unless the `--auto-approve` or `-y` flag is used.

**Confirmation Behavior:**
- By default, you will be prompted with `Are you sure you want to delete {resource_type} {id}? [y/N]:`
- Type `y` or `yes` to confirm deletion
- Type `n`, `no`, or press Enter to cancel
- Use `--auto-approve` or `-y` to skip the confirmation prompt (useful for automation)

#### `ucctl delete object <id>`

Delete an AuthZ object by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Delete Flags:**
- `-y, --auto-approve` - Automatically approve deletion without prompting

**Examples:**

```bash
# Interactive deletion with confirmation prompt
ucctl ctx use tenant-foo
ucctl delete object 550e8400-e29b-41d4-a716-446655440000
# Prompts: Are you sure you want to delete object 550e8400-e29b-41d4-a716-446655440000? [y/N]:

# Auto-approve (no prompt) - useful for scripts
ucctl delete object 550e8400-e29b-41d4-a716-446655440000 -y

# Auto-approve with long flag
ucctl delete object 550e8400-e29b-41d4-a716-446655440000 --auto-approve
```

#### `ucctl delete objecttype <id>`

Delete an AuthZ object type by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Delete Flags:**
- `-y, --auto-approve` - Automatically approve deletion without prompting

**Examples:**

```bash
# Interactive deletion with confirmation prompt
ucctl delete objecttype 550e8400-e29b-41d4-a716-446655440000

# Auto-approve (no prompt)
ucctl delete objecttype 550e8400-e29b-41d4-a716-446655440000 -y
```

**Note:** Cannot delete object types that have existing objects. Delete all objects first.

#### `ucctl delete edge <id>`

Delete an AuthZ edge by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Delete Flags:**
- `-y, --auto-approve` - Automatically approve deletion without prompting

**Examples:**

```bash
# Interactive deletion with confirmation prompt
ucctl delete edge 550e8400-e29b-41d4-a716-446655440000

# Auto-approve (no prompt)
ucctl delete edge 550e8400-e29b-41d4-a716-446655440000 -y
```

#### `ucctl delete edgetype <id>`

Delete an AuthZ edge type by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Delete Flags:**
- `-y, --auto-approve` - Automatically approve deletion without prompting

**Examples:**

```bash
# Interactive deletion with confirmation prompt
ucctl delete edgetype 550e8400-e29b-41d4-a716-446655440000

# Auto-approve (no prompt)
ucctl delete edgetype 550e8400-e29b-41d4-a716-446655440000 -y
```

**Note:** Cannot delete edge types that have existing edges. Delete all edges of this type first.

---

### Set Commands

Commands for setting properties on UserClouds resources.

**Important Note on Resource Immutability:**
- **Object Types** and **Edges** are immutable after creation and cannot be updated
- Only **Objects** (via alias) and **Edge Types** (via all fields) support update operations

#### `ucctl set object <id>`

Update an AuthZ object's alias field by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Update Flags (must specify one):**
- `-a, --alias` - New alias for the object
- `--clear-alias` - Clear the alias (set to null)

**Examples:**

Set an alias:
```bash
ucctl ctx use tenant-foo
ucctl set object 550e8400-e29b-41d4-a716-446655440000 --alias "user-admin"
```

Clear an alias:
```bash
ucctl set object 550e8400-e29b-41d4-a716-446655440000 --clear-alias
```

**Notes:**
- Only the alias field can be updated on objects
- Other object fields (type_id, organization_id) are immutable
- You must specify either `--alias` or `--clear-alias`, but not both

#### `ucctl set edgetype <id>`

Update an AuthZ edge type by its ID.

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**Update Flags (all required):**
- `--type-name` - New type name for the edge type
- `--source-object-type-id` - Source object type ID (UUID)
- `--target-object-type-id` - Target object type ID (UUID)

**Optional Flags:**
- `--attributes` - JSON string of edge type attributes

**Examples:**

Update edge type with basic fields:
```bash
ucctl ctx use tenant-foo
ucctl set edgetype 550e8400-e29b-41d4-a716-446655440000 \
  --type-name "member" \
  --source-object-type-id 660e8400-e29b-41d4-a716-446655440001 \
  --target-object-type-id 770e8400-e29b-41d4-a716-446655440002 \
 
```

Update edge type with attributes:
```bash
ucctl set edgetype 550e8400-e29b-41d4-a716-446655440000 \
  --type-name "member" \
  --source-object-type-id 660e8400-e29b-41d4-a716-446655440001 \
  --target-object-type-id 770e8400-e29b-41d4-a716-446655440002 \
  --attributes '{"role":"admin","permissions":["read","write"]}' \
 
```

**Notes:**
- All edge type fields are updatable
- All required flags must be provided (even if not changing)
- Attributes must be valid JSON if provided
- Time-based fields (created, updated) are managed automatically

#### `ucctl set admin`

Set admin privileges for a user on a tenant or company.

**Important:** The command uses the current context:
- **For tenant operations**: Switch to the tenant context first
- **For company operations**: Switch to the console tenant context first

**Connection Flags (override context):**
- `--url` - Tenant URL (overrides context)
- `--client-id` - OAuth2 client ID (overrides context)
- `--client-secret` - OAuth2 client secret (overrides context)
- `--client-secret-var` - Environment variable containing client secret (default: `UC_CLIENT_SECRET`)

**User Identification (must specify one):**
- `-e, --email` - User email address
- `-u, --user-id` - User ID (UUID)

**Target (must specify one):**
- `-t, --tenant-id` - Tenant ID (UUID) to grant admin privileges on
- `-c, --company-id` - Company ID (UUID) to grant admin privileges on

**Other Flags:**
- `-v, --verbose` - Enable verbose output

**Examples:**

**Grant admin privileges on a tenant:**

```bash
# Switch to the tenant context
ucctl ctx use tenant-foo

# Grant admin using email
ucctl set admin \
  --email user@example.com \
  --tenant-id 550e8400-e29b-41d4-a716-446655440000

# Or using user ID
ucctl set admin \
  --user-id 660e8400-e29b-41d4-a716-446655440002 \
  --tenant-id 550e8400-e29b-41d4-a716-446655440000 \
  --verbose
```

**Grant admin privileges on a company:**

```bash
# Switch to the console tenant context (companies are managed through console)
ucctl ctx use console-prod

# Grant admin on company
ucctl set admin \
  --email user@example.com \
  --company-id 550e8400-e29b-41d4-a716-446655440001
```

**Using explicit connection flags:**

```bash
ucctl set admin \
  --url https://foo.tenant.example.com \
  --client-id my-client \
  --client-secret my-secret \
  --email user@example.com \
  --tenant-id 550e8400-e29b-41d4-a716-446655440000
```

**Notes:**
- **For tenant operations**: The command connects to the tenant's authz system using the current context
- **For company operations**: The command connects to the console tenant's authz system (companies are managed through the console tenant)
- When using `--email`, the command searches for the user by email address (tries OIDC first, then password auth)
- If multiple users have the same email, you must use `--user-id` instead
- The command adds the user to the target tenant/company group with the `admin` role

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
ucctl ctx set staging --url https://staging.tenant.example.com --client-id staging-client --client-secret staging-secret
ucctl ctx set prod --url https://prod.tenant.example.com --client-id prod-client --client-secret prod-secret

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
  --source-url https://staging.tenant.example.com \
  --source-client-id staging-client \
  --destination-url https://prod.tenant.example.com \
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
current_context: console-prod
contexts:
  console-prod:
    url: https://console.tenant.example.com
    client_id: console-client
    client_secret: console-secret
    console_tenant: true
  tenant-foo:
    url: https://foo.tenant.example.com
    client_id: foo-client
    client_secret: foo-secret
    console: console-prod
  tenant-bar:
    url: https://bar.tenant.example.com
    client_id: bar-client
    client_secret: bar-secret
    console: console-prod
```

**Field Descriptions:**
- `url`: The base URL for the UserClouds installation
- `client_id`: OAuth2 client ID for authentication
- `client_secret`: OAuth2 client secret for authentication
- `console`: Name of the console tenant context (for regular tenant contexts only)
- `console_tenant`: Boolean flag indicating this is a console tenant (for console contexts only)

**Security Note:** The config file contains client secrets in plain text. Ensure the file has appropriate permissions (mode 0600).

---

## Environment Variables

- `UC_CLIENT_SECRET` - Default environment variable for client secrets (used by `create user` and `sync tenant` commands)
- Custom environment variables can be specified via `--client-secret-var` and `--source-client-secret-var`/`--destination-client-secret-var` flags

---

## Common Workflows

### Setting Up Console and Tenant Contexts

```bash
# First, create the console tenant context
ucctl ctx set console-prod \
  --url https://console.tenant.example.com \
  --client-id console-client \
  --client-secret console-secret \
  --console-tenant

# Create tenant contexts that reference the console
ucctl ctx set tenant-foo \
  --url https://foo.tenant.example.com \
  --client-id foo-client \
  --client-secret foo-secret \
  --console console-prod

ucctl ctx set tenant-bar \
  --url https://bar.tenant.example.com \
  --client-id bar-client \
  --client-secret bar-secret \
  --console console-prod

# List all contexts
ucctl ctx list

# Switch between contexts
ucctl ctx use tenant-foo
```

### Creating Console Employees and Managing Companies

```bash
# Switch to console tenant context
ucctl ctx use console-prod

# Create a console employee
ucctl create user \
  --username admin \
  --password AdminPass123! \
  --email admin@example.com \
  --name "Admin User" \
  --organization-id <console-org-id>

# Grant admin privileges on a company (still in console context)
ucctl set admin \
  --email admin@example.com \
  --company-id <company-uuid>
```

### Creating and Managing Tenant Users

```bash
# Switch to tenant context
ucctl ctx use tenant-foo

# Create a tenant user
ucctl create user \
  --username customer1 \
  --password CustomerPass123! \
  --email customer@example.com \
  --name "Customer User" \
  --organization-id <tenant-org-id>

# Grant admin privileges on the tenant (still in tenant context)
ucctl set admin \
  --email customer@example.com \
  --tenant-id <tenant-uuid>
```

### Syncing Configuration Between Environments

```bash
# Using contexts (recommended)
ucctl ctx set staging --url https://staging.tenant.example.com --client-id staging-client --client-secret staging-secret
ucctl ctx set local --url https://local.tenant.example.com --client-id local-client --client-secret local-secret

# Dry run: preview what would change
ucctl sync tenant --source staging --destination local --dry-run --verbose

# Apply the sync
ucctl sync tenant --source staging --destination local

# Or using explicit URLs (without contexts)
export UC_CLIENT_SECRET=staging-secret
export UC_DEST_SECRET=local-secret
ucctl sync tenant \
  --source-url https://staging.tenant.example.com \
  --source-client-id staging-client \
  --destination-url https://local.tenant.example.com \
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
# Create a console tenant context
ucctl ctx set console-prod \
  --url https://console.tenant.example.com \
  --client-id console-client \
  --client-secret console-secret \
  --console-tenant

# Or create a tenant context (requires a console tenant context first)
ucctl ctx set tenant-foo \
  --url https://foo.tenant.example.com \
  --client-id foo-client \
  --client-secret foo-secret \
  --console console-prod

# Switch to the context
ucctl ctx use console-prod

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

## License

See the root repository LICENSE file for licensing information.
