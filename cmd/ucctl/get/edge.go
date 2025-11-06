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

type EdgeCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	// credentials holds the loaded credentials
	credentials *common.Credentials
}

func (c *EdgeCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("edge ID is required")
	}

	edgeID, err := uuid.FromString(args[0])
	if err != nil {
		return fmt.Errorf("invalid edge ID: %w", err)
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

	// Get the edge
	edge, err := azClient.GetEdge(cmd.Context(), edgeID)
	if err != nil {
		return fmt.Errorf("failed to get edge: %w", err)
	}

	// Display the edge
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tSOURCE_OBJECT_ID\tTARGET_OBJECT_ID\tEDGE_TYPE_ID\tCREATED\tUPDATED")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		edge.ID.String(),
		edge.SourceObjectID.String(),
		edge.TargetObjectID.String(),
		edge.EdgeTypeID.String(),
		edge.Created.Format("2006-01-02 15:04:05"),
		edge.Updated.Format("2006-01-02 15:04:05"),
	)

	return w.Flush()
}
