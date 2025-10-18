package get

import (
	"fmt"
	"github.com/gofrs/uuid"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/config"
	"userclouds.com/idp"
	"userclouds.com/infra/jsonclient"
)

type UserCommand struct {
	UseContext   bool
	URL          string
	ClientID     string
	ClientSecret string
	AuthnType    string
}

func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	emailOrUuid := args[0]

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

	authType := idp.AuthnTypeAll
	switch c.AuthnType {
	case "social":
		authType = idp.AuthnTypeOIDC
	case "password":
		authType = idp.AuthnTypePassword
	}

	var profiles []idp.UserBaseProfileAndAuthnResponse

	// Check to see if the argument is a uuid.
	if id, err := uuid.FromString(emailOrUuid); err == nil {
		fmt.Println("here")
		p, err := mgmtClient.GetUserBaseProfileAndAuthN(cmd.Context(), id)
		if err != nil {
			return fmt.Errorf("failed to get user base profile and authn: %w", err)
		}
		profiles = append(profiles, *p)
	} else {
		p, err := mgmtClient.ListUserBaseProfilesAndAuthNForEmail(cmd.Context(), emailOrUuid, authType)
		if err != nil {
			return err
		}
		profiles = append(profiles, p...)
	}

	fmt.Println("profiles", profiles)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tORGANIZATION\tVERIFIED\tAUTHN TYPE\tPROVIDER")

	for _, profile := range profiles {
		if len(profile.Authns) > 0 {
			for _, authn := range profile.Authns {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\t%s\n", profile.ID, profile.Name, profile.Email, profile.OrganizationID, profile.EmailVerified, authn.AuthnType, authn.OIDCProvider)
			}
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\t%s\n", profile.ID, profile.Name, profile.Email, profile.OrganizationID, profile.EmailVerified, "", "")
		}
		// TODO: Maybe show MFA channels?
	}

	return w.Flush()
}

func printWithAuthn(w io.Writer, profile *idp.UserBaseProfileAndAuthnResponse) {
	for _, authn := range profile.Authns {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\t%s\n", profile.ID, profile.Name, profile.Email, profile.OrganizationID, profile.EmailVerified, authn.AuthnType, authn.OIDCProvider)
	}
}

func (c *UserCommand) getClientCredentials() (jsonclient.Option, error) {
	return jsonclient.ClientCredentialsForURL(c.URL, c.ClientID, c.ClientSecret, nil)
}
