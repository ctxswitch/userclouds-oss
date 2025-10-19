package delete

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/ucerr"
)

type ObjectCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	AutoApprove     bool

	credentials *common.Credentials
}

func (c *ObjectCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("object ID is required")
	}

	objectID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid object ID: %w", err)
	}

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

	// Create client credentials option
	credOpt, err := c.credentials.GetClientCredentials()
	if err != nil {
		return fmt.Errorf("failed to create client credentials: %w", err)
	}

	// Create AuthZ client
	azClient, err := authz.NewClient(c.credentials.URL, authz.JSONClient(credOpt))
	if err != nil {
		return fmt.Errorf("failed to create AuthZ client: %w", err)
	}

	// Prompt for confirmation
	if !common.ConfirmDeletion("object", objectID.String(), c.AutoApprove) {
		fmt.Println("Deletion cancelled")
		return nil
	}

	// Delete the object
	err = azClient.DeleteObject(cmd.Context(), objectID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	fmt.Printf("Object %s deleted successfully\n", objectID)
	return nil
}
