package get

import (
	"fmt"
	"os"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
)

type ObjectTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	// credentials holds the loaded credentials
	credentials *common.Credentials
}

func (c *ObjectTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ucctl get objecttype <id>")
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

	// Get object type
	objectType, err := azClient.GetObjectType(cmd.Context(), objectTypeID)
	if err != nil {
		return fmt.Errorf("failed to get object type: %w", err)
	}

	// Display result
	fmt.Fprintf(os.Stdout, "Object Type Details:\n")
	fmt.Fprintf(os.Stdout, "  ID:         %s\n", objectType.ID.String())
	fmt.Fprintf(os.Stdout, "  Type Name:  %s\n", objectType.TypeName)
	fmt.Fprintf(os.Stdout, "  Created:    %s\n", objectType.Created.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(os.Stdout, "  Updated:    %s\n", objectType.Updated.Format("2006-01-02 15:04:05"))

	return nil
}
