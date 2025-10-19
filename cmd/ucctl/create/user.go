package create

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/oidc"
)

// UserCommand handles the create user subcommand
type UserCommand struct {
	Admin           bool
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	OrganizationID  string
	Email           string
	Name            string
	Username        string
	Password        string
	OIDCProvider    string
	OIDCIssuerURL   string
	OIDCSubject     string
	Verbose         bool

	// credentials holds the loaded credentials
	credentials *common.Credentials
}

// RunE executes the create user command
func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if err := c.validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := c.createUser(ctx); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (c *UserCommand) validate() error {
	// Organization is required.
	if c.OrganizationID == "" {
		return fmt.Errorf("organization id is required")
	}

	// Load credentials from context or flags
	creds, err := common.LoadCredentialsFromContext(
		
		c.URL,
		c.ClientID,
		c.ClientSecret,
		c.ClientSecretVar,
	)
	if err != nil {
		return err
	}
	c.credentials = creds

	// Validate authentication method (optional - can create user without authn)
	hasPassword := c.Username != "" && c.Password != ""
	hasOIDC := c.OIDCProvider != "" && c.OIDCIssuerURL != "" && c.OIDCSubject != ""

	// Validate partial OIDC parameters
	hasPartialOIDC := c.OIDCProvider != "" || c.OIDCIssuerURL != "" || c.OIDCSubject != ""
	if hasPartialOIDC && !hasOIDC {
		return fmt.Errorf("when using OIDC authentication, all three flags are required: --oidc-provider, --oidc-issuer-url, and --oidc-subject")
	}

	// Validate partial password parameters
	hasPartialPassword := c.Username != "" || c.Password != ""
	if hasPartialPassword && !hasPassword {
		return fmt.Errorf("when using password authentication, both --username and --password are required")
	}

	if hasPassword && hasOIDC {
		return fmt.Errorf("cannot provide both password and OIDC authentication - choose one or omit both to create user without authentication")
	}

	return nil
}

func (c *UserCommand) createUser(ctx context.Context) error {
	spinner, _ := pterm.DefaultSpinner.Start("Initializing...")

	// Create client credentials option
	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		spinner.Fail("Failed to create client credentials")
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	// Create IDP management client
	mgmtClient, err := idp.NewManagementClient(c.credentials.URL, credOpt)
	if err != nil {
		spinner.Fail("Failed to create IDP client")
		return fmt.Errorf("failed to create IDP client: %w", err)
	}

	// Build user profile
	profile := userstore.Record{}
	if c.Email != "" {
		profile["email"] = c.Email
	}
	if c.Name != "" {
		profile["name"] = c.Name
	}

	var opts []idp.Option
	orgID, err := uuid.FromString(c.OrganizationID)
	if err != nil {
		spinner.Fail("Invalid organization ID")
		return fmt.Errorf("invalid organization ID: %w", err)
	}
	opts = append(opts, idp.OrganizationID(orgID))

	var userID uuid.UUID

	// Create user with appropriate authn method (or without authn)
	if c.Username != "" && c.Password != "" {
		spinner.UpdateText("Creating user with password authentication...")
		userID, err = mgmtClient.CreateUserWithPassword(ctx, c.Username, c.Password, profile, opts...)
		if err != nil {
			spinner.Fail("Failed to create user with password")
			return fmt.Errorf("failed to create user with password: %w", err)
		}
	} else if c.OIDCProvider != "" {
		var provider oidc.ProviderType
		if err := provider.UnmarshalText([]byte(c.OIDCProvider)); err != nil {
			spinner.Fail("Invalid OIDC provider")
			return fmt.Errorf("invalid OIDC provider: %w", err)
		}

		spinner.UpdateText("Creating user with OIDC authentication...")
		userID, err = mgmtClient.CreateUserWithOIDC(ctx, provider, c.OIDCIssuerURL, c.OIDCSubject, profile, opts...)
		if err != nil {
			spinner.Fail("Failed to create user with OIDC")
			return fmt.Errorf("failed to create user with OIDC: %w", err)
		}
	} else {
		spinner.UpdateText("Creating user without authentication...")
		userID, err = mgmtClient.CreateUser(ctx, profile, opts...)
		if err != nil {
			spinner.Fail("Failed to create user")
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	spinner.Success("User created successfully")

	if c.Verbose {
		pterm.Println()
		pterm.DefaultBox.WithTitle("User Created").WithTitleTopCenter().Println(
			pterm.Sprintf("User ID: %s\nEmail: %s\nName: %s",
				pterm.LightCyan(userID.String()),
				pterm.LightCyan(c.Email),
				pterm.LightCyan(c.Name)),
		)
	}

	// TODO: Allow for other policies.
	if c.Admin {
		err := c.setAdmin(ctx, userID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *UserCommand) setAdmin(ctx context.Context, id uuid.UUID) error {
	return nil
}
