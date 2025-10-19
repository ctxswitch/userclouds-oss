package get

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/idp"
)

type UserCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string

	// credentials holds the loaded credentials
	credentials *common.Credentials
}

func (c *UserCommand) RunE(cmd *cobra.Command, args []string) error {
	emailOrUuid := args[0]

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
