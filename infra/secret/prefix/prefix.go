package prefix

import (
	"net/url"
	"strings"
)

type PrefixError string

func (e PrefixError) Error() string {
	return string(e)
}

const (
	ErrorPrefixInvalid PrefixError = "invalid prefix"
)

// Prefix is just a type alias to make some automation easier
type Prefix string

const (
	// PrefixAWS tells secret that this string is in fact resolvable with AWS secret manager
	// as opposed to other systems in the future, or just plaintext (for eg. dev)
	// TODO: config linter in the future that ensures all secret.* fields are prefixed in prod configs?
	PrefixAWS Prefix = "aws://secrets/"
	// PrefixDev tells us this is a dev-only Base64 encoded secret
	PrefixDev Prefix = "dev://"
	// PrefixDevLiteral tells us this is dev-only and not obfuscated
	// This exists separate from DevPrefix because sometimes it's useful
	// to be able to read the secret in plaintext (eg. in ci.yaml files)
	// and sometimes we want a slightly more interesting test of resolvers
	PrefixDevLiteral Prefix = "dev-literal://"
	// PrefixKubernetes informs that the secret is a kubernetes secret.
	PrefixKubernetes Prefix = "kube://secrets/"
	// PrefixEnv tells us this is a secret from the environment variables
	PrefixEnv Prefix = "env://"
)

//go:generate genconstant Prefix

// Matches is a poorly named function to check if a string starts with a prefix
func (p Prefix) Matches(s string) bool {
	return strings.HasPrefix(s, string(p))
}

// Value gets the non-prefixed value of a string, stripping any query parameters
func (p Prefix) Value(s string) string {
	value := strings.TrimPrefix(s, string(p))
	// Strip query parameters if present
	if idx := strings.Index(value, "?"); idx != -1 {
		value = value[:idx]
	}
	return value
}

// ValueWithParams gets the non-prefixed value of a string, preserving query parameters
func (p Prefix) ValueWithParams(s string) string {
	return strings.TrimPrefix(s, string(p))
}

// ParseParams extracts query parameters from a prefixed string
// Returns the path and the parsed query parameters
func (p Prefix) ParseParams(s string) (string, url.Values, error) {
	value := strings.TrimPrefix(s, string(p))

	// Check if there are query parameters
	if idx := strings.Index(value, "?"); idx != -1 {
		path := value[:idx]
		queryString := value[idx+1:]
		params, err := url.ParseQuery(queryString)
		if err != nil {
			return "", nil, err
		}
		return path, params, nil
	}

	// No query parameters
	return value, url.Values{}, nil
}

func (p Prefix) String() string {
	return string(p)
}

func PrefixFromString(s string) (Prefix, error) {
	for _, prefix := range AllPrefixes {
		if strings.HasPrefix(s, string(prefix)) {
			return prefix, nil
		}
	}

	return "", ErrorPrefixInvalid
}
