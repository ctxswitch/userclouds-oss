package main

import (
	"github.com/spf13/cobra"

	"userclouds.com/cmd/ucctl/context"
	"userclouds.com/cmd/ucctl/create"
	"userclouds.com/cmd/ucctl/sync"
)

const (
	RootUsage       = "ucctl"
	RootShort       = "CLI utility for interacting with userclouds"
	RootLong        = `CLI utility for interacting with userclouds`
	SyncUsage       = "sync"
	SyncShort       = "Sync resources between environments"
	SyncLong        = `Sync UserClouds resources between different environments`
	CreateUsage     = "create"
	CreateShort     = "Create userclouds resources"
	CreateLong      = `Create userclouds resources`
	CreateUserUsage = "user"
	CreateUserShort = "Create a new user with or without authentication"
	CreateUserLong  = `Create a new user with password authentication, OIDC authentication, or without authentication.

When creating a user without authentication, the authentication will be automatically
added when the user logs in for the first time via OIDC (based on email match).`
	ContextUsage = "context"
	ContextShort = "Manage userclouds contexts"
	ContextLong  = `Manage userclouds contexts for different environments`
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

	rootCmd.AddCommand(SyncCommand())
	rootCmd.AddCommand(CreateCommand())
	rootCmd.AddCommand(ContextCommand())
	return rootCmd
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

func CreateUserCommand() *cobra.Command {
	uc := create.UserCommand{}
	cmd := &cobra.Command{
		Use:   CreateUserUsage,
		Short: CreateUserShort,
		Long:  CreateUserLong,
		RunE:  uc.RunE,
	}

	cmd.Flags().BoolVarP(&uc.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().BoolVarP(&uc.UseContext, "use-context", "", false, "use current context from config")
	cmd.Flags().StringVarP(&uc.URL, "url", "", "", "IDP URL (or use context)")
	cmd.Flags().StringVarP(&uc.ClientID, "client-id", "", "", "client ID (or use context)")
	cmd.Flags().StringVarP(&uc.ClientSecret, "client-secret", "", "", "client secret (or use context)")
	cmd.Flags().StringVarP(&uc.ClientSecretVar, "client-secret-var", "", create.DefaultClientSecretVar, "environment variable containing client secret")
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
		Long:  "Create or update a UserClouds context configuration",
		RunE:  sc.RunE,
	}

	cmd.Flags().StringVarP(&sc.URL, "url", "", "", "UserClouds URL (required)")
	cmd.Flags().StringVarP(&sc.ClientID, "client-id", "", "", "OAuth2 client ID (required)")
	cmd.Flags().StringVarP(&sc.ClientSecret, "client-secret", "", "", "OAuth2 client secret (required)")

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
