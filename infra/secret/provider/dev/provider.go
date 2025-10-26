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

type Provider struct {
	decode bool
}

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

func (p *Provider) Prefix() string {
	// This is a bit naive, but at the moment dev only enables decode.
	if p.decode {
		return PrefixDev
	}

	return PrefixDevLiteral
}

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
