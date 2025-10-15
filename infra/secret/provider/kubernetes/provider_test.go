package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetes_pathToSecretName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			"simple",
			"dummysecret",
			"dummysecret",
		},
		{"token path",
			"cross_service_auth_token/161a09be-5c61-4c76-a084-bcfe9b4968c2",
			"cross_service_auth_token.161a09be-5c61-4c76-a084-bcfe9b4968c2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := pathToSecretName(tt.path)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestKubernetes_Get(t *testing.T) {
	tests := []struct {
		path  string
		value string
	}{
		{"dummy-service", "super_secret"},
		{"dummy-service/dummy-secret", "really_super_secret"},
	}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			client := fake.NewSimpleClientset(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pathToSecretName(tt.path),
					Namespace: DefaultNamespace,
				},
				Data: map[string][]byte{
					"value": []byte(tt.value),
				},
			})

			provider := New().WithClient(client)
			value, err := provider.Get(ctx, tt.path)
			assert.NoError(t, err)
			assert.Equal(t, tt.value, string(value))
		})
	}
}

func TestKubernetes_Delete(t *testing.T) {
	ctx := context.Background()

	client := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy-service",
			Namespace: DefaultNamespace,
		},
		Data: map[string][]byte{
			"value": []byte("super_secret"),
		},
	})

	provider := New().WithClient(client)
	err := provider.Delete(ctx, "dummy-service")
	assert.NoError(t, err)

	_, err = client.CoreV1().Secrets(DefaultNamespace).Get(ctx, "dummy-service", metav1.GetOptions{})
	assert.True(t, errors.IsNotFound(err))
}

func TestKubernetes_Save(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	provider := New().WithClient(client)

	err := provider.Save(ctx, "dummy-service", "super_secret")
	assert.NoError(t, err)

	secret, err := client.CoreV1().Secrets(DefaultNamespace).Get(ctx, "dummy-service", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "super_secret", string(secret.Data["value"]))

	err = provider.Save(ctx, "dummy-service", "really_super_secret")
	assert.NoError(t, err)
	secret, err = client.CoreV1().Secrets(DefaultNamespace).Get(ctx, "dummy-service", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "really_super_secret", string(secret.Data["value"]))
}
