package get

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
)

type EdgeTypeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	// credentials holds the loaded credentials
	credentials *common.Credentials
}

func (c *EdgeTypeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
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

	// Get the edge type
	edgeType, err := azClient.GetEdgeType(cmd.Context(), edgeTypeID)
	if err != nil {
		return fmt.Errorf("failed to get edge type: %w", err)
	}

	// Display the edge type
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE_NAME\tSOURCE_TYPE_ID\tTARGET_TYPE_ID\tORGANIZATION_ID\tCREATED\tUPDATED")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		edgeType.ID.String(),
		edgeType.TypeName,
		edgeType.SourceObjectTypeID.String(),
		edgeType.TargetObjectTypeID.String(),
		edgeType.OrganizationID.String(),
		edgeType.Created.Format("2006-01-02 15:04:05"),
		edgeType.Updated.Format("2006-01-02 15:04:05"),
	)

	return w.Flush()
}
