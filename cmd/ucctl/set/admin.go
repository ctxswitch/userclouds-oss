package set

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
)

// AdminCommand handles the set admin subcommand
type AdminCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	UserEmail       string
	UserID          string
	TenantID        string
	CompanyID       string
	Verbose         bool

	// credentials holds the loaded credentials
	credentials *common.Credentials
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
	// Load credentials from context or flags
	creds, err := common.LoadCredentialsFromContext(
		
		
		c.URL,
		c.ClientID,
		c.ClientSecret,
		c.ClientSecretVar,
		"", // configPath - use default precedence
	)
	if err != nil {
		return err
	}
	c.credentials = creds

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
	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		return uuid.Nil, err
	}

	// Create IDP management client
	mgmtClient, err := idp.NewManagementClient(c.credentials.URL, credOpt)
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

func (c *AdminCommand) setAdminForTenant(ctx context.Context, spinner *pterm.SpinnerPrinter, userID uuid.UUID) error {
	spinner.UpdateText("Setting admin privileges for tenant...")

	tenantID, err := uuid.FromString(c.TenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	// Get client credentials
	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	// Create authz client
	authzClient, err := authz.NewClient(c.credentials.URL, authz.JSONClient(credOpt), authz.TenantID(tenantID))
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

	// Get client credentials
	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	// NOTE: Companies are managed through the console tenant's authz system.
	// The current context should be set to the console tenant context.

	// Create authz client connected to console tenant
	authzClient, err := authz.NewClient(c.credentials.URL, authz.JSONClient(credOpt))
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
