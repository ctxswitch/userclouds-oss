package prefix

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPrefix_Validate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid aws", "aws://secrets/", true},
		{"invalid aws", "aws://secrets", false},
		{"invalid aws string", "aws://not-a-secret/", false},
		{"valid kubernetes", "kube://secrets/", true},
		{"invalid kubernetes", "kube://secrets", false},
		{"invalid kubernetes string", "kube://not-a-secret/", false},
		{"invalid", "not-a-secret", false},
		{"valid env", "env://", true},
		{"valid dev", "dev://", true},
		{"valid dev literal", "dev-literal://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := Prefix(tt.input)
			if tt.valid {
				assert.NoError(t, prefix.Validate())
			} else {
				assert.Error(t, prefix.Validate())
			}
		})
	}
}

func TestPrefix_Values(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{"aws secret path", "aws://secrets/my-secret", "my-secret"},
		{"kubernetes secret path", "kube://secrets/my-secret", "my-secret"},
		{"env secret path", "env://my-secret", "my-secret"},
		{"dev secret path", "dev://my-secret", "my-secret"},
		{"dev-literal secret path", "dev-literal://my-secret", "my-secret"},
		{"longer path", "aws://secrets/path-to/my-secret", "path-to/my-secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: This is a bit redundant?
			prefix, err := PrefixFromString(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.output, prefix.Value(tt.input))
		})
	}
}

func TestPrefix_PrefixFromString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output Prefix
	}{
		{"aws secret", "aws://secrets/my-secret", PrefixAWS},
		{"kubernetes secret", "kube://secrets/my-secret", PrefixKubernetes},
		{"env secret", "env://my-secret", PrefixEnv},
		{"dev secret", "dev://my-secret", PrefixDev},
		{"dev-literal", "dev-literal://my-secret", PrefixDevLiteral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, err := PrefixFromString(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.output, prefix)
		})
	}
}
