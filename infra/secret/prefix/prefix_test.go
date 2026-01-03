package prefix

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestPrefix_ParseParams(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedPath   string
		expectedParams map[string]string
		expectError    bool
	}{
		{
			name:           "no query params",
			input:          "aws://secrets/my-secret",
			expectedPath:   "my-secret",
			expectedParams: map[string]string{},
			expectError:    false,
		},
		{
			name:         "single query param",
			input:        "aws://secrets/my-secret?profile=production",
			expectedPath: "my-secret",
			expectedParams: map[string]string{
				"profile": "production",
			},
			expectError: false,
		},
		{
			name:         "multiple query params",
			input:        "aws://secrets/my-secret?profile=production&region=us-west-2",
			expectedPath: "my-secret",
			expectedParams: map[string]string{
				"profile": "production",
				"region":  "us-west-2",
			},
			expectError: false,
		},
		{
			name:           "path with slashes and query params",
			input:          "aws://secrets/path/to/my-secret?profile=prod",
			expectedPath:   "path/to/my-secret",
			expectedParams: map[string]string{"profile": "prod"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, err := PrefixFromString(tt.input)
			assert.NoError(t, err)

			path, params, err := prefix.ParseParams(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, path)
				for key, expectedValue := range tt.expectedParams {
					assert.Equal(t, expectedValue, params.Get(key))
				}
			}
		})
	}
}

func TestPrefix_Value_WithQueryParams(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
	}{
		{
			name:          "no query params",
			input:         "aws://secrets/my-secret",
			expectedValue: "my-secret",
		},
		{
			name:          "with query params strips them",
			input:         "aws://secrets/my-secret?profile=production",
			expectedValue: "my-secret",
		},
		{
			name:          "multiple query params strips them",
			input:         "aws://secrets/my-secret?profile=prod&region=us-west-2",
			expectedValue: "my-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, err := PrefixFromString(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedValue, prefix.Value(tt.input))
		})
	}
}

func TestPrefix_ParseParams_InvalidQueryString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "invalid query string format",
			input:       "aws://secrets/my-secret?profile=prod&invalid",
			expectError: false, // url.ParseQuery handles this gracefully
		},
		{
			name:        "empty value is valid",
			input:       "aws://secrets/my-secret?profile=",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, err := PrefixFromString(tt.input)
			assert.NoError(t, err)

			_, _, err = prefix.ParseParams(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
