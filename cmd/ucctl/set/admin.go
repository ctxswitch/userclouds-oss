package set

import (
	"context"
	"fmt"
	"os"

	"github.com/gofrs/uuid"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/cmd/ucctl/config"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
)

// AdminCommand handles the set admin subcommand
type AdminCommand struct {
	URL                       string
	ConsoleTenantURL          string
	ClientID                  string
	ClientSecret              string
	ConsoleTenantClientID     string
	ConsoleTenantClientSecret string
	ClientSecretVar           string
	UseContext                bool
	UserEmail                 string
	UserID                    string
	TenantID                  string
	CompanyID                 string
	Verbose                   bool
}

const (
	DefaultClientSecretVar = "UC_CLIENT_SECRET"
)

// NewAdminCommand creates the set admin command
func NewAdminCommand() *cobra.Command {
	ac := &AdminCommand{}
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Set admin privileges for a user",
		Long: `Set admin privileges for a user on a tenant or company.
The user can be specified by email or ID.
Either --tenant-id or --company-id must be specified.`,
		RunE: ac.RunE,
	}

	cmd.Flags().BoolVarP(&ac.Verbose, "verbose", "v", false, "verbose output")
	cmd.Flags().BoolVarP(&ac.UseContext, "use-context", "", false, "use current context from config")
	cmd.Flags().StringVarP(&ac.URL, "url", "", "", "IDP URL (or use context)")
	cmd.Flags().StringVarP(&ac.ConsoleTenantURL, "console-tenant-url", "", "", "Console tenant URL (or use context, required for company operations)")
	cmd.Flags().StringVarP(&ac.ClientID, "client-id", "", "", "client ID (or use context)")
	cmd.Flags().StringVarP(&ac.ClientSecret, "client-secret", "", "", "client secret (or use context)")
	cmd.Flags().StringVarP(&ac.ClientSecretVar, "client-secret-var", "", DefaultClientSecretVar, "environment variable containing client secret")
	cmd.Flags().StringVarP(&ac.UserEmail, "email", "e", "", "user email address")
	cmd.Flags().StringVarP(&ac.UserID, "user-id", "u", "", "user ID")
	cmd.Flags().StringVarP(&ac.TenantID, "tenant-id", "t", "", "tenant ID")
	cmd.Flags().StringVarP(&ac.CompanyID, "company-id", "c", "", "company ID")

	return cmd
}

// RunE executes the set admin command
func (c *AdminCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if err := c.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := c.setAdmin(ctx); err != nil {
		return fmt.Errorf("failed to set admin privileges: %w", err)
	}

	return nil
}

func (c *AdminCommand) validate() error {
	// If using context, load from config
	if c.UseContext || (c.URL == "" && c.ClientID == "" && c.ClientSecret == "") {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ctx, err := cfg.GetCurrentContext()
		if err != nil {
			return fmt.Errorf("no current context set. Use 'ucctl context use <name>' or provide --url, --client-id, and --client-secret")
		}

		// Only override if not explicitly set
		if c.URL == "" {
			c.URL = ctx.URL
		}
		if c.ClientID == "" {
			c.ClientID = ctx.ClientID
		}
		if c.ClientSecret == "" {
			c.ClientSecret = ctx.ClientSecret
		}

		// Resolve console tenant context for console credentials
		consoleCtx, err := cfg.GetConsoleTenantContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve console tenant: %w", err)
		}

		if c.ConsoleTenantURL == "" {
			c.ConsoleTenantURL = consoleCtx.URL
		}
		if c.ConsoleTenantClientID == "" {
			c.ConsoleTenantClientID = consoleCtx.ClientID
		}
		if c.ConsoleTenantClientSecret == "" {
			c.ConsoleTenantClientSecret = consoleCtx.ClientSecret
		}
	}

	if c.URL == "" {
		return fmt.Errorf("URL is required (use --url or set a context)")
	}

	if c.ClientID == "" {
		return fmt.Errorf("client ID is required (use --client-id or set a context)")
	}

	// Get client secret from environment variable if not set directly
	if c.ClientSecret == "" {
		c.ClientSecret = os.Getenv(c.ClientSecretVar)
		if c.ClientSecret == "" {
			return fmt.Errorf("client secret is required (use --client-secret, set %s env var, or use a context)", c.ClientSecretVar)
		}
	}

	// User must be specified by either email or ID
	if c.UserEmail == "" && c.UserID == "" {
		return fmt.Errorf("user must be specified by either --email or --user-id")
	}

	if c.UserEmail != "" && c.UserID != "" {
		return fmt.Errorf("specify only one of --email or --user-id, not both")
	}

	// Must specify either tenant or company
	if c.TenantID == "" && c.CompanyID == "" {
		return fmt.Errorf("either --tenant-id or --company-id must be specified")
	}

	if c.TenantID != "" && c.CompanyID != "" {
		return fmt.Errorf("specify only one of --tenant-id or --company-id, not both")
	}

	// For company operations, console tenant URL is required
	if c.CompanyID != "" && c.ConsoleTenantURL == "" {
		return fmt.Errorf("--console-tenant-url is required when setting admin for a company (companies are managed through the console tenant)")
	}

	return nil
}

func (c *AdminCommand) setAdmin(ctx context.Context) error {
	spinner, _ := pterm.DefaultSpinner.Start("Initializing...")

	// Get the user ID if email was provided
	var userID uuid.UUID
	var err error

	if c.UserEmail != "" {
		userID, err = c.getUserIDByEmail(ctx)
		if err != nil {
			spinner.Fail("Failed to find user by email")
			return fmt.Errorf("failed to find user by email: %w", err)
		}
		if c.Verbose {
			pterm.Info.Printf("Found user with email %s: %s\n", c.UserEmail, userID)
		}
	} else {
		userID, err = uuid.FromString(c.UserID)
		if err != nil {
			spinner.Fail("Invalid user ID")
			return fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// Set admin based on tenant or company
	if c.TenantID != "" {
		err = c.setAdminForTenant(ctx, spinner, userID)
	} else {
		err = c.setAdminForCompany(ctx, spinner, userID)
	}

	if err != nil {
		spinner.Fail("Failed to set admin privileges")
		return err
	}

	spinner.Success("Admin privileges set successfully")

	if c.Verbose {
		pterm.Println()
		targetType := "Tenant"
		targetID := c.TenantID
		if c.CompanyID != "" {
			targetType = "Company"
			targetID = c.CompanyID
		}
		pterm.DefaultBox.WithTitle("Admin Privileges Set").WithTitleTopCenter().Println(
			pterm.Sprintf("User ID: %s\n%s ID: %s\nRole: %s",
				pterm.LightCyan(userID.String()),
				targetType,
				pterm.LightCyan(targetID),
				pterm.LightCyan(ucauthz.AdminRole)),
		)
	}

	return nil
}

func (c *AdminCommand) getUserIDByEmail(ctx context.Context) (uuid.UUID, error) {
	// Get client credentials
	credOpt, err := c.getClientCredentials()
	if err != nil {
		return uuid.Nil, err
	}

	// Create IDP management client
	mgmtClient, err := idp.NewManagementClient(c.URL, credOpt)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create IDP client: %w", err)
	}

	// Search for users by email
	users, err := mgmtClient.ListUserBaseProfilesAndAuthNForEmail(ctx, c.UserEmail, idp.AuthnTypeOIDC)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to search for user: %w", err)
	}

	if len(users) == 0 {
		// Try password auth as well
		users, err = mgmtClient.ListUserBaseProfilesAndAuthNForEmail(ctx, c.UserEmail, idp.AuthnTypePassword)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to search for user: %w", err)
		}
	}

	if len(users) == 0 {
		return uuid.Nil, fmt.Errorf("no user found with email: %s", c.UserEmail)
	}

	if len(users) > 1 {
		return uuid.Nil, fmt.Errorf("multiple users found with email: %s, please use --user-id instead", c.UserEmail)
	}

	userID, err := uuid.FromString(users[0].ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID from server: %w", err)
	}

	return userID, nil
}

func (c *AdminCommand) getClientCredentials() (jsonclient.Option, error) {
	return jsonclient.ClientCredentialsForURL(c.URL, c.ClientID, c.ClientSecret, nil)
}

func (c *AdminCommand) getConsoleTenantClientCredentials() (jsonclient.Option, error) {
	return jsonclient.ClientCredentialsForURL(c.ConsoleTenantURL, c.ConsoleTenantClientID, c.ConsoleTenantClientSecret, nil)
}

func (c *AdminCommand) setAdminForTenant(ctx context.Context, spinner *pterm.SpinnerPrinter, userID uuid.UUID) error {
	spinner.UpdateText("Setting admin privileges for tenant...")

	tenantID, err := uuid.FromString(c.TenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	// Get client credentials
	credOpt, err := c.getClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	// Create authz client
	authzClient, err := authz.NewClient(c.URL, authz.JSONClient(credOpt), authz.TenantID(tenantID))
	if err != nil {
		return fmt.Errorf("failed to create authz client: %w", err)
	}

	rbacClient := authz.NewRBACClient(authzClient)

	// Get the user object
	user, err := rbacClient.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get the group (use tenant ID as group ID)
	group, err := rbacClient.GetGroup(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant group: %w", err)
	}

	// Add user to group with admin role
	_, err = group.AddUserRole(ctx, *user, ucauthz.AdminRole)
	if err != nil {
		return fmt.Errorf("failed to add admin role: %w", err)
	}

	return nil
}

func (c *AdminCommand) setAdminForCompany(ctx context.Context, spinner *pterm.SpinnerPrinter, userID uuid.UUID) error {
	spinner.UpdateText("Setting admin privileges for company...")

	companyID, err := uuid.FromString(c.CompanyID)
	if err != nil {
		return fmt.Errorf("invalid company ID: %w", err)
	}

	// Get console tenant client credentials
	credOpt, err := c.getConsoleTenantClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create console tenant client credentials: %w", err)
	}

	// NOTE: Companies are managed through the console tenant's authz system.
	// The ConsoleTenantURL should point to the console tenant, not the company itself.
	// The console tenant manages company memberships and roles.

	// Create authz client connected to console tenant
	authzClient, err := authz.NewClient(c.ConsoleTenantURL, authz.JSONClient(credOpt))
	if err != nil {
		return fmt.Errorf("failed to create authz client: %w", err)
	}

	rbacClient := authz.NewRBACClient(authzClient)

	// Get the user (user must exist in console tenant)
	user, err := rbacClient.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user in console tenant: %w", err)
	}

	// Get the company group (company is a group in console tenant)
	group, err := rbacClient.GetGroup(ctx, companyID)
	if err != nil {
		return fmt.Errorf("failed to get company group: %w", err)
	}

	// Add user to company with admin role
	_, err = group.AddUserRole(ctx, *user, ucauthz.AdminRole)
	if err != nil {
		return fmt.Errorf("failed to add admin role: %w", err)
	}

	return nil
}
