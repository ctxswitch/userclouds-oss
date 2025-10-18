package get

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/config"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

type UsersCommand struct {
	URL          string
	ClientID     string
	ClientSecret string
}

func (c *UsersCommand) RunE(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cx, err := cfg.GetCurrentContext()
	if err != nil {
		return fmt.Errorf("no current context set. Use 'ucctl context use <name>' or provide --url, --client-id, and --client-secret")
	}

	c.URL = cx.URL
	c.ClientID = cx.ClientID
	c.ClientSecret = cx.ClientSecret

	credOpt, err := c.getClientCredentials()
	if err != nil {
		return err
	}

	// Create IDP management client
	mgmtClient, err := idp.NewManagementClient(c.URL, credOpt)
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

func (c *UsersCommand) getClientCredentials() (jsonclient.Option, error) {
	return jsonclient.ClientCredentialsForURL(c.URL, c.ClientID, c.ClientSecret, nil)
}
