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

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Prefix() string {
	return Prefix
}

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
