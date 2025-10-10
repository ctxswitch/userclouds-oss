package synctenant

import (
	"fmt"
	"net/url"
	"os"
	"userclouds.com/authz"
	"userclouds.com/infra/jsonclient"
)

type tenant struct {
	tenantURL       string
	clientID        string
	clientSecretVar string
	authzURL        *url.URL
	tokenSource     jsonclient.Option
}

func NewTenant(url string, clientID string, clientSecretVar string) *tenant {
	return &tenant{
		tenantURL:       url,
		clientID:        clientID,
		clientSecretVar: clientSecretVar,
	}
}

func (t *tenant) GetClient() (*authz.Client, error) {
	if err := t.initToken(); err != nil {
		return nil, err
	}

	return authz.NewClient(t.tenantURL, authz.JSONClient(t.tokenSource))
}

func (t *tenant) initToken() error {
	authzURL, err := url.Parse(t.tenantURL)
	if err != nil {
		return fmt.Errorf("unable to parse tenant URL %s: %v", t.tenantURL, err)
	}

	secret := os.Getenv(t.clientSecretVar)

	ts, err := jsonclient.ClientCredentialsForURL(t.tenantURL, t.clientID, secret, nil)
	if err != nil {
		return fmt.Errorf("failed to create token source for %s: %v", t.tenantURL, err)
	}

	t.authzURL = authzURL
	t.tokenSource = ts

	return nil
}
