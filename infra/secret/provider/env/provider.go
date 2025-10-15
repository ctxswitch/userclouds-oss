package env

import (
	"context"
	"os"
	"regexp"

	"userclouds.com/infra/ucerr"
)

const (
	Prefix = "env://"
)

var specialCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// Provider defines a new secrets provider.
type Provider struct{}

// New returns a new environment variable based secrets provider.
func New() *Provider {
	return &Provider{}
}

// Prefix returns the URI prefix for an environment variable based secret.
func (p *Provider) Prefix() string {
	return Prefix
}

// IsDev is a helper function that returns true if the provider is explicitly used
// in development environments.  This allows for dev specific behaviors to be handled.
func (p *Provider) IsDev() bool {
	return false
}

// Get returns a secret from an environment variable.
func (p *Provider) Get(ctx context.Context, path string) (string, error) {
	secret, defined := os.LookupEnv(path)
	if !defined {
		return "", ucerr.Errorf("Can't load secret from environment variable %s", path)
	}

	if secret == "" {
		return "", ucerr.Errorf("Secret from environment variable %s is empty", path)
	}

	return secret, nil
}

// Save does nothing for the env provider.
func (p *Provider) Save(ctx context.Context, path, secret string) error {
	return nil
}

// Delete does nothing for the env provider.
func (p *Provider) Delete(ctx context.Context, path string) error {
	return nil
}
