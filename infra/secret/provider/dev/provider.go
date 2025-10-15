package dev

import (
	"context"
	"encoding/base64"

	"userclouds.com/infra/ucerr"
)

const (
	PrefixDev        = "dev://"
	PrefixDevLiteral = "dev-literal://"
)

// Provider defines a development provider.
type Provider struct {
	decode bool
}

// New returns an initialized development provider.
func New() *Provider {
	return &Provider{
		decode: true,
	}
}

// WithLiterals enables base64 decoding of the secret.
func (p *Provider) WithLiterals() *Provider {
	p.decode = false
	return p
}

// Prefix returns the URI prefix for an inline development secret.  The dev provider
// can take two forms.  The first, as dev which contains a base64 encoded secret and
// second as dev-literal which contains an unencoded plain text secret.  These are used
// in testing and should not be used in production services.
func (p *Provider) Prefix() string {
	// This is a bit naive, but at the moment dev only enables decode.
	if p.decode {
		return PrefixDev
	}

	return PrefixDevLiteral
}

// IsDev is a helper function that returns true if the provider is explicitly used
// in development environments.  This allows for dev specific behaviors to be handled.
func (p *Provider) IsDev() bool {
	return true
}

// Get is just a passthrough returning the 'path' which is the secret value
// i.e. dev://<base64_encoded_secret> or dev-literal://<secret>.
func (p *Provider) Get(ctx context.Context, path string) (string, error) {
	secret := path

	if p.decode {
		bs, err := base64.StdEncoding.DecodeString(secret)
		if err != nil {
			return "", ucerr.Wrap(err)
		}

		secret = string(bs)
	}

	return secret, nil
}

// Save does nothing for the dev provider.
func (p *Provider) Save(ctx context.Context, path, secret string) error {
	return nil
}

// Delete does nothing for the dev provider.
func (p *Provider) Delete(ctx context.Context, path string) error {
	return nil
}
