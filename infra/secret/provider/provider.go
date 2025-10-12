package provider

import (
	"context"
	"fmt"
	"os"

	"userclouds.com/infra/secret/prefix"
	"userclouds.com/infra/secret/provider/aws"
	"userclouds.com/infra/secret/provider/dev"
	"userclouds.com/infra/secret/provider/env"
	"userclouds.com/infra/secret/provider/kubernetes"
)

const (
	SecretManagerEnvKey = "UC_SECRET_MANAGER"
)

type Interface interface {
	Get(ctx context.Context, path string) (string, error)
	Delete(ctx context.Context, path string) error
	Save(ctx context.Context, path, secret string) error
	Prefix() string
}

type Provider struct{}

// FromEnv returns the discovered provider.  There are three that are supported
// currently: 'aws', 'kube', and 'dev'.  This is not the best way to manage this.
// I'd like to merge into the config at a later time, but this is the most straight
// forward approach given how it is handled right now (based on universe env vars)
// since there would need to be other changes to the callers.
func FromEnv() Interface {
	// Supporting three stores at the moment.  If the store isn't defined we choose the
	// expected AWS for cloud and on-prem universes.  I may get rid of `dev` later on since
	// the local development environment is also changing.
	value, isDefined := os.LookupEnv(SecretManagerEnvKey)
	if !isDefined {
		return aws.New()
	}

	storeMap := map[string]Interface{
		"aws":        aws.New(),
		"kubernetes": kubernetes.New(),
		"dev":        dev.New(),
		"env":        env.New(),
	}

	if provider, found := storeMap[value]; found {
		return provider
	}

	// Assume AWS provider for all unknown.  I don't particularly like this, but there are some
	// the overall behavior previously was to prefer AWS, so keep this.  This is a blind error
	// though so might want to at least log here, but didn't want to pass in the context.
	return aws.New()
}

func FromLocation(loc string) (Interface, error) {
	// Get the prefix
	px, err := prefix.PrefixFromString(loc)
	if err != nil {
		return nil, err
	}

	switch px {
	case prefix.PrefixAWS:
		return aws.New(), nil
	case prefix.PrefixEnv:
		return env.New(), nil
	case prefix.PrefixKubernetes:
		return kubernetes.New(), nil
	case prefix.PrefixDev:
		return dev.New().WithDecodeEnabled(), nil
	case prefix.PrefixDevLiteral:
		return dev.New(), nil
	}

	return nil, fmt.Errorf("unknown secret provider for %s", loc)
}
