package get

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

type UsersCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	// credentials holds the loaded credentials
	credentials *common.Credentials
}

func (c *UsersCommand) RunE(cmd *cobra.Command, args []string) error {
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

	// Create IDP management client
	mgmtClient, err := idp.NewManagementClient(c.credentials.URL, credOpt)
	if err != nil {
		return fmt.Errorf("failed to create IDP client: %w", err)
	}

	profiles, err := c.fetchProfiles(cmd.Context(), mgmtClient)
	if err != nil {
		return fmt.Errorf("failed to list user base profiles: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tORGANIZATION\tVERIFIED")

	for _, profile := range profiles {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\n", profile.ID, profile.Name, profile.Email, profile.OrganizationID, profile.EmailVerified)
	}

	return w.Flush()
}

func (c *UsersCommand) fetchProfiles(ctx context.Context, mgc *idp.ManagementClient) ([]idp.UserBaseProfileResponse, error) {
	var profiles []idp.UserBaseProfileResponse
	cursor := pagination.CursorBegin

	for {
		resp, err := mgc.ListUserBaseProfiles(ctx, idp.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		profiles = append(profiles, resp.Data...)
		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}

	return profiles, nil
}
