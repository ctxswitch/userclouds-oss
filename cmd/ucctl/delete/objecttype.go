package delete

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/ucerr"
)

type ObjectTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	AutoApprove     bool

	credentials *common.Credentials
}

func (c *ObjectTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("object type ID is required")
	}

	objectTypeID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid object type ID: %w", err)
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
	if !common.ConfirmDeletion("object type", objectTypeID.String(), c.AutoApprove) {
		fmt.Println("Deletion cancelled")
		return nil
	}

	// Delete the object type
	err = azClient.DeleteObjectType(cmd.Context(), objectTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	fmt.Printf("Object type %s deleted successfully\n", objectTypeID)
	return nil
}
