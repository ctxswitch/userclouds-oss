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

type ObjectCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	// credentials holds the loaded credentials
	credentials *common.Credentials
}

func (c *ObjectCommand) RunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
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

	// Get the object
	obj, err := azClient.GetObject(cmd.Context(), objectID)
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}

	var alias string
	if obj.Alias != nil {
		alias = *obj.Alias
	}

	// Display the object
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tALIAS\tTYPE_ID\tORGANIZATION_ID\tCREATED\tUPDATED")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		obj.ID.String(),
		alias,
		obj.TypeID.String(),
		obj.OrganizationID.String(),
		obj.Created.Format("2006-01-02 15:04:05"),
		obj.Updated.Format("2006-01-02 15:04:05"),
	)

	return w.Flush()
}
