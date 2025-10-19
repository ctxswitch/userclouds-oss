package main

import (
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/autoprovision"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/cmd/ucctl/get"

	"userclouds.com/cmd/ucctl/context"
	"userclouds.com/cmd/ucctl/create"
	"userclouds.com/cmd/ucctl/delete"
	"userclouds.com/cmd/ucctl/set"
	"userclouds.com/cmd/ucctl/sync"
)

const (
	RootUsage          = "ucctl"
	RootShort          = "CLI utility for interacting with userclouds"
	RootLong           = `CLI utility for interacting with userclouds`
	AutoProvisionUsage = "autoprovision"
	AutoProvisionShort = "Autoprovision the console tenant"
	AutoProvisionLong  = `Autoprovision creates a base company and the console tenant from a template file`
	SyncUsage          = "sync"
	SyncShort          = "Sync resources between environments"
	SyncLong           = `Sync UserClouds resources between different environments`
	CreateUsage        = "create"
	CreateShort        = "Create userclouds resources"
	CreateLong         = `Create userclouds resources`
	CreateUserUsage    = "user"
	CreateUserShort    = "Create a new user"
	CreateUserLong     = `Create a new user with password authentication, OIDC authentication, or without authentication.

The user is created in the tenant specified by the current context.
- For console employee users: set context to a console tenant context
- For regular tenant users: set context to a regular tenant context

When creating a user without authentication, the authentication will be automatically
added when the user logs in for the first time via OIDC (based on email match).`
	ContextUsage = "context"
	ContextShort = "Manage userclouds contexts"
	ContextLong  = `Manage userclouds contexts for different environments`
	SetUsage     = "set"
	SetShort     = "Set userclouds resources"
	SetLong      = `Set userclouds resources such as admin privileges`
	GetUsage     = "get"
	GetShort     = "Get userclouds resources"
	GetLong      = `Get userclouds resources such as users, objects, edges, etc`
	DeleteUsage  = "delete"
	DeleteShort  = "Delete userclouds resources"
	DeleteLong   = `Delete userclouds resources such as objects, edges, object types, and edge types`
)

type Root struct{}

func NewRoot() *Root {
	return &Root{}
}

func (r *Root) Execute() error {
	return r.Command().Execute()
}

func (r *Root) Command() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   RootUsage,
		Short: RootShort,
		Long:  RootLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	rootCmd.AddCommand(AutoProvisionCommand())
	rootCmd.AddCommand(SyncCommand())
	rootCmd.AddCommand(CreateCommand())
	rootCmd.AddCommand(DeleteCommand())
	rootCmd.AddCommand(SetCommand())
	rootCmd.AddCommand(ContextCommand())
	rootCmd.AddCommand(GetCommand())
	return rootCmd
}

func AutoProvisionCommand() *cobra.Command {
	ap := autoprovision.AutoProvisionCommand{}
	cmd := &cobra.Command{
		Use:   AutoProvisionUsage,
		Short: AutoProvisionShort,
		Long:  AutoProvisionLong,
		RunE:  ap.RunE,
	}

	return cmd
}

func SyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   SyncUsage,
		Short: SyncShort,
		Long:  SyncLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	// TODO: Add more sync subcommands (tokenizer, userstore, authn, logserver)
	cmd.AddCommand(sync.NewTenantCommand())
	return cmd
}

func CreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   CreateUsage,
		Short: CreateShort,
		Long:  CreateLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(CreateUserCommand())
	return cmd
}

func GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   GetUsage,
		Short: GetShort,
		Long:  GetLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(GetUsersCommand())
	cmd.AddCommand(GetUserCommand())
	cmd.AddCommand(GetObjectTypesCommand())
	cmd.AddCommand(GetObjectTypeCommand())
	cmd.AddCommand(GetObjectsCommand())
	cmd.AddCommand(GetObjectCommand())
	cmd.AddCommand(GetEdgeTypesCommand())
	cmd.AddCommand(GetEdgeTypeCommand())
	cmd.AddCommand(GetEdgesCommand())
	cmd.AddCommand(GetEdgeCommand())
	return cmd
}

func DeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   DeleteUsage,
		Short: DeleteShort,
		Long:  DeleteLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(DeleteObjectCommand())
	cmd.AddCommand(DeleteObjectTypeCommand())
	cmd.AddCommand(DeleteEdgeCommand())
	cmd.AddCommand(DeleteEdgeTypeCommand())
	return cmd
}

func SetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   SetUsage,
		Short: SetShort,
		Long:  SetLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(SetAdminCommand())
	cmd.AddCommand(SetObjectCommand())
	cmd.AddCommand(SetEdgeTypeCommand())
	return cmd
}

func CreateUserCommand() *cobra.Command {
	uc := create.UserCommand{}
	cmd := &cobra.Command{
		Use:   CreateUserUsage,
		Short: CreateUserShort,
		Long:  CreateUserLong,
		RunE:  uc.RunE,
	}

	common.AddAuthFlags(cmd, &uc.URL, &uc.ClientID, &uc.ClientSecret, &uc.ClientSecretVar, &uc.AuthnType)

	cmd.Flags().BoolVarP(&uc.Admin, "admin", "a", false, "Admin user")
	cmd.Flags().BoolVarP(&uc.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&uc.OrganizationID, "organization-id", "", "", "organization ID for the user")
	cmd.Flags().StringVarP(&uc.Email, "email", "", "", "user email address")
	cmd.Flags().StringVarP(&uc.Name, "name", "", "", "user name")
	cmd.Flags().StringVarP(&uc.Username, "username", "", "", "username for password authentication")
	cmd.Flags().StringVarP(&uc.Password, "password", "", "", "password for password authentication")
	cmd.Flags().StringVarP(&uc.OIDCProvider, "oidc-provider", "", "", "OIDC provider (e.g., google, github)")
	cmd.Flags().StringVarP(&uc.OIDCIssuerURL, "oidc-issuer-url", "", "", "OIDC issuer URL")
	cmd.Flags().StringVarP(&uc.OIDCSubject, "oidc-subject", "", "", "OIDC subject ID")

	return cmd
}

func ContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     ContextUsage,
		Aliases: []string{"ctx"},
		Short:   ContextShort,
		Long:    ContextLong,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(ContextListCommand())
	cmd.AddCommand(ContextUseCommand())
	cmd.AddCommand(ContextSetCommand())
	cmd.AddCommand(ContextDeleteCommand())
	cmd.AddCommand(ContextShowCommand())
	return cmd
}

func ContextListCommand() *cobra.Command {
	lc := context.ListCommand{}
	return &cobra.Command{
		Use:   "list",
		Short: "List all contexts",
		Long:  "List all configured UserClouds contexts",
		RunE:  lc.RunE,
	}
}

func ContextUseCommand() *cobra.Command {
	uc := context.UseCommand{}
	return &cobra.Command{
		Use:   "use <context-name>",
		Short: "Switch to a context",
		Long:  "Switch to a different UserClouds context",
		RunE:  uc.RunE,
	}
}

func ContextSetCommand() *cobra.Command {
	sc := context.SetCommand{}
	cmd := &cobra.Command{
		Use:   "set <context-name>",
		Short: "Create or update a context",
		Long: `Create or update a UserClouds context configuration.

For regular tenant contexts, specify --console to reference a console tenant.
For console tenant contexts, use the --console-tenant flag.`,
		RunE: sc.RunE,
	}

	cmd.Flags().StringVarP(&sc.URL, "url", "", "", "UserClouds URL (required)")
	cmd.Flags().StringVarP(&sc.ClientID, "client-id", "", "", "OAuth2 client ID (required)")
	cmd.Flags().StringVarP(&sc.ClientSecret, "client-secret", "", "", "OAuth2 client secret (required)")
	cmd.Flags().StringVarP(&sc.Console, "console", "", "", "Console tenant context name (required for tenant contexts)")
	cmd.Flags().BoolVarP(&sc.IsConsoleTenant, "console-tenant", "", false, "Mark this context as a console tenant")

	return cmd
}

func ContextDeleteCommand() *cobra.Command {
	dc := context.DeleteCommand{}
	return &cobra.Command{
		Use:   "delete <context-name>",
		Short: "Delete a context",
		Long:  "Delete a UserClouds context configuration",
		RunE:  dc.RunE,
	}
}

func ContextShowCommand() *cobra.Command {
	sc := context.ShowCommand{}
	return &cobra.Command{
		Use:   "show",
		Short: "Show current context",
		Long:  "Display the currently active UserClouds context",
		RunE:  sc.RunE,
	}
}

func GetUsersCommand() *cobra.Command {
	gu := get.UsersCommand{}
	cmd := &cobra.Command{
		Use:   "users",
		Short: "List all users",
		Long:  "List all users",
		RunE:  gu.RunE,
	}

	common.AddAuthFlags(cmd, &gu.URL, &gu.ClientID, &gu.ClientSecret, &gu.ClientSecretVar, &gu.AuthnType)

	return cmd
}

func GetUserCommand() *cobra.Command {
	gu := get.UserCommand{}
	cmd := &cobra.Command{
		Use:   "user <email_or_uuid>",
		Short: "Get user profiles by email",
		Long:  "Get user profiles by email",
		RunE:  gu.RunE,
	}

	common.AddAuthFlags(cmd, &gu.URL, &gu.ClientID, &gu.ClientSecret, &gu.ClientSecretVar, &gu.AuthnType)

	return cmd
}

func GetObjectTypesCommand() *cobra.Command {
	got := get.ObjectTypesCommand{}
	cmd := &cobra.Command{
		Use:   "objecttypes",
		Short: "List all object types",
		Long:  "List all AuthZ object types with pagination support",
		RunE:  got.RunE,
	}

	common.AddListCommandFlags(cmd, &got.URL, &got.ClientID, &got.ClientSecret, &got.ClientSecretVar, &got.Limit, &got.Cursor, &got.NoPager, &got.Filter, &got.RawFilter, &got.Output, &got.AuthnType)

	return cmd
}

func GetObjectTypeCommand() *cobra.Command {
	got := get.ObjectTypeCommand{}
	cmd := &cobra.Command{
		Use:   "objecttype <id>",
		Short: "Get object type by ID",
		Long:  "Get detailed information about an AuthZ object type by its ID",
		RunE:  got.RunE,
	}

	common.AddAuthFlags(cmd, &got.URL, &got.ClientID, &got.ClientSecret, &got.ClientSecretVar, &got.AuthnType)

	return cmd
}

func GetObjectsCommand() *cobra.Command {
	gobj := get.ObjectsCommand{}
	cmd := &cobra.Command{
		Use:   "objects",
		Short: "List all objects",
		Long:  "List all AuthZ objects with pagination and filtering support",
		RunE:  gobj.RunE,
	}

	common.AddListCommandFlags(cmd, &gobj.URL, &gobj.ClientID, &gobj.ClientSecret, &gobj.ClientSecretVar, &gobj.Limit, &gobj.Cursor, &gobj.NoPager, &gobj.Filter, &gobj.RawFilter, &gobj.Output, &gobj.AuthnType)

	return cmd
}

func GetObjectCommand() *cobra.Command {
	gobj := get.ObjectCommand{}
	cmd := &cobra.Command{
		Use:   "object <id>",
		Short: "Get object by ID",
		Long:  "Get detailed information about an AuthZ object by its ID",
		RunE:  gobj.RunE,
	}

	common.AddAuthFlags(cmd, &gobj.URL, &gobj.ClientID, &gobj.ClientSecret, &gobj.ClientSecretVar, &gobj.AuthnType)

	return cmd
}

func GetEdgeTypesCommand() *cobra.Command {
	ge := get.EdgeTypesCommand{}
	cmd := &cobra.Command{
		Use:   "edgetypes",
		Short: "List all edge types",
		Long:  "List all AuthZ edge types with pagination and filtering support",
		RunE:  ge.RunE,
	}

	common.AddListCommandFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.Limit, &ge.Cursor, &ge.NoPager, &ge.Filter, &ge.RawFilter, &ge.Output, &ge.AuthnType)

	return cmd
}

func GetEdgeTypeCommand() *cobra.Command {
	ge := get.EdgeTypeCommand{}
	cmd := &cobra.Command{
		Use:   "edgetype <id>",
		Short: "Get edge type by ID",
		Long:  "Get detailed information about an AuthZ edge type by its ID",
		RunE:  ge.RunE,
	}

	common.AddAuthFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.AuthnType)

	return cmd
}

func GetEdgesCommand() *cobra.Command {
	ge := get.EdgesCommand{}
	cmd := &cobra.Command{
		Use:   "edges",
		Short: "List all edges",
		Long:  "List all AuthZ edges with pagination and filtering support",
		RunE:  ge.RunE,
	}

	common.AddListCommandFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.Limit, &ge.Cursor, &ge.NoPager, &ge.Filter, &ge.RawFilter, &ge.Output, &ge.AuthnType)

	return cmd
}

func GetEdgeCommand() *cobra.Command {
	ge := get.EdgeCommand{}
	cmd := &cobra.Command{
		Use:   "edge <id>",
		Short: "Get edge by ID",
		Long:  "Get detailed information about an AuthZ edge by its ID",
		RunE:  ge.RunE,
	}

	common.AddAuthFlags(cmd, &ge.URL, &ge.ClientID, &ge.ClientSecret, &ge.ClientSecretVar, &ge.AuthnType)

	return cmd
}

func DeleteObjectCommand() *cobra.Command {
	dc := delete.ObjectCommand{}
	cmd := &cobra.Command{
		Use:   "object <id>",
		Short: "Delete an object by ID",
		Long:  "Delete an AuthZ object by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)

	return cmd
}

func DeleteObjectTypeCommand() *cobra.Command {
	dc := delete.ObjectTypeCommand{}
	cmd := &cobra.Command{
		Use:   "objecttype <id>",
		Short: "Delete an object type by ID",
		Long:  "Delete an AuthZ object type by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)

	return cmd
}

func DeleteEdgeCommand() *cobra.Command {
	dc := delete.EdgeCommand{}
	cmd := &cobra.Command{
		Use:   "edge <id>",
		Short: "Delete an edge by ID",
		Long:  "Delete an AuthZ edge by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)

	return cmd
}

func DeleteEdgeTypeCommand() *cobra.Command {
	dc := delete.EdgeTypeCommand{}
	cmd := &cobra.Command{
		Use:   "edgetype <id>",
		Short: "Delete an edge type by ID",
		Long:  "Delete an AuthZ edge type by its ID",
		RunE:  dc.RunE,
	}

	common.AddAuthFlags(cmd, &dc.URL, &dc.ClientID, &dc.ClientSecret, &dc.ClientSecretVar, &dc.AuthnType)

	return cmd
}

func SetObjectCommand() *cobra.Command {
	sc := set.ObjectCommand{}
	cmd := &cobra.Command{
		Use:   "object <id>",
		Short: "Update an object's alias",
		Long:  "Update an AuthZ object's alias field by its ID",
		RunE:  sc.RunE,
	}

	common.AddAuthFlags(cmd, &sc.URL, &sc.ClientID, &sc.ClientSecret, &sc.ClientSecretVar, &sc.AuthnType)

	cmd.Flags().StringVarP(&sc.Alias, "alias", "a", "", "new alias for the object")
	cmd.Flags().BoolVarP(&sc.ClearAlias, "clear-alias", "", false, "clear the alias (set to null)")

	return cmd
}

func SetEdgeTypeCommand() *cobra.Command {
	sc := set.EdgeTypeCommand{}
	cmd := &cobra.Command{
		Use:   "edgetype <id>",
		Short: "Update an edge type",
		Long:  "Update an AuthZ edge type's properties by its ID",
		RunE:  sc.RunE,
	}

	common.AddAuthFlags(cmd, &sc.URL, &sc.ClientID, &sc.ClientSecret, &sc.ClientSecretVar, &sc.AuthnType)

	cmd.Flags().StringVarP(&sc.TypeName, "type-name", "n", "", "edge type name (required)")
	cmd.Flags().StringVarP(&sc.SourceObjectTypeID, "source-object-type-id", "s", "", "source object type ID (required)")
	cmd.Flags().StringVarP(&sc.TargetObjectTypeID, "target-object-type-id", "t", "", "target object type ID (required)")
	cmd.Flags().StringVarP(&sc.Attributes, "attributes", "", "", "attributes as JSON string")

	return cmd
}

func SetAdminCommand() *cobra.Command {
	ac := set.AdminCommand{}
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Set admin privileges for a user",
		Long: `Set admin privileges for a user on a tenant or company.

The user can be specified by email or ID.
Either --tenant-id or --company-id must be specified.

For tenant operations: Switch to the tenant context first
For company operations: Switch to the console tenant context first`,
		RunE: ac.RunE,
	}

	common.AddAuthFlags(cmd, &ac.URL, &ac.ClientID, &ac.ClientSecret, &ac.ClientSecretVar, &ac.AuthnType)

	cmd.Flags().BoolVarP(&ac.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().StringVarP(&ac.UserEmail, "email", "e", "", "user email address")
	cmd.Flags().StringVarP(&ac.UserID, "user-id", "u", "", "user ID")
	cmd.Flags().StringVarP(&ac.TenantID, "tenant-id", "t", "", "tenant ID")
	cmd.Flags().StringVarP(&ac.CompanyID, "company-id", "c", "", "company ID")

	return cmd
}
