package secret

import (
	"context"
	"encoding/json"
	"testing"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
	"userclouds.com/infra/secret/provider"
	"userclouds.com/infra/secret/provider/aws"
	"userclouds.com/infra/secret/provider/kubernetes"
)

type testStruct struct {
	Secret *String `yaml:"secret" json:"secret" db:"secret"`
}

func TestFieldYAML(t *testing.T) {
	ctx := context.Background()
	y := "secret: dev-literal://not-actually-secret"
	var got testStruct
	assert.NoError(t, yaml.Unmarshal([]byte(y), &got))
	s, err := got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "not-actually-secret")
	assert.Equal(t, got.Secret.String(), "*********************************") // NB: .String() is required for types to match in assert

	y = "secret: dev://Zm9v"
	got = testStruct{}
	assert.NoError(t, yaml.Unmarshal([]byte(y), &got))
	s, err = got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "foo")
}

func TestFieldJSON(t *testing.T) {
	ctx := context.Background()
	j := `{"secret":"dev-literal://testme"}`
	var got testStruct
	assert.NoError(t, json.Unmarshal([]byte(j), &got))
	s, err := got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "testme")
	assert.Equal(t, got.Secret.String(), "********************") // NB: .String() is required for types to match in assert

	j = `{"secret":"dev://Zm9v"}`
	got = testStruct{}
	assert.NoError(t, json.Unmarshal([]byte(j), &got))
	s, err = got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "foo")
	assert.Equal(t, got.Secret.String(), "**********") // NB: .String() is required for types to match in assert

	// make sure we only round trip the location
	bs, err := json.Marshal(got)
	assert.NoError(t, err)
	assert.Equal(t, string(bs), j)
}

func TestFromLocation(t *testing.T) {
	ctx := context.Background()
	devSecret := FromLocation("dev-literal://festivus")
	sv, err := devSecret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, sv, "festivus")
	assert.Equal(t, devSecret.String(), "**********************") // NB: .String() is required for types to match in assert
}

func TestField_Value(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{"testsecret", "testsecret", "testsecret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &String{location: tt.input}
			val, err := s.Value()
			assert.NoError(t, err)
			assert.Equal(t, val, tt.output)
		})
	}
}

func TestField_Scan(t *testing.T) {
	tests := []struct {
		name  string
		input any
		valid bool
	}{
		{"string", "testsecret", true},
		{"int", 1, false},
		{"bool", true, false},
		{"complex", struct{ name string }{"name"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &String{}
			err := s.Scan(tt.input)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestField_Validate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"testsecret", "testsecret", false},
		{"empty", "", true},
		{"aws://secrets/testsecret", "aws://secrets/testsecret", true},
		{"kube://secrets/testsecret", "kube://secrets/testsecret", true},
		{"aws://not-a-secret/testsecret", "aws://not-a-secret/testsecret", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &String{
				location: tt.input,
			}
			if tt.valid {
				assert.NoError(t, s.Validate())
			} else {
				assert.Error(t, s.Validate())
			}
		})
	}
}

func TestField_NewString_Dev(t *testing.T) {
	tests := []struct {
		description string
		service     string
		name        string
		value       string
		location    string
	}{
		{"simple", "service", "my-secret", "testsecret", "dev://dGVzdHNlY3JldA=="},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			ctx := context.Background()

			t.Setenv(provider.SecretManagerEnvKey, "dev")
			t.Setenv("UC_UNIVERSE", "test")

			s, err := NewString(ctx, tt.service, tt.name, tt.value)
			assert.NoError(t, err)
			assert.Equal(t, tt.location, s.location)
		})
	}
}

func TestField_Resolve(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
		setup  func(*String)
	}{
		{"dev", "dev://bXktc2VjcmV0", "my-secret", func(*String) {}},
		{"dev-literal", "dev-literal://my-secret", "my-secret", func(*String) {}},
		{"env", "env://MY_SECRET", "my-secret", func(*String) {
			t.Setenv("MY_SECRET", "my-secret")
		}},
		{"kubernetes", "kube://secrets/my-secret", "testsecret", func(s *String) {
			s.WithProvider(kubernetes.New().WithClient(fake.NewSimpleClientset(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: kubernetes.DefaultNamespace,
				},
				Data: map[string][]byte{
					"value": []byte("testsecret"),
				},
			})))
		}},
		{"aws", "aws://secrets/my-secret", "testsecret", func(s *String) {
			sm := &aws.MockSecretsManagerClient{}
			input := &secretsmanager.GetSecretValueInput{
				SecretId:     ptr.To("my-secret"),
				VersionStage: ptr.To("AWSCURRENT"),
			}
			sm.On("GetSecretValue", mock.Anything, input, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
				SecretString: awsv2.String(`{"string":"testsecret"}`),
			}, nil)
			s.WithProvider(aws.New().WithSecretsManagerClient(sm))
		}},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := FromLocation(tt.input)

			tt.setup(s)
			secret, err := s.Resolve(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.output, secret)

			cachedValue, found := c.Get(tt.input)
			assert.True(t, found)
			assert.Equal(t, tt.output, cachedValue)
		})
	}
}
