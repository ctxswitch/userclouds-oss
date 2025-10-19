package delete

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/ucerr"
)

type EdgeTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	credentials *common.Credentials
}

func (c *EdgeTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("edge type ID is required")
	}

	edgeTypeID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid edge type ID: %w", err)
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

	// Delete the edge type
	err = azClient.DeleteEdgeType(cmd.Context(), edgeTypeID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	fmt.Printf("Edge type %s deleted successfully\n", edgeTypeID)
	return nil
}
