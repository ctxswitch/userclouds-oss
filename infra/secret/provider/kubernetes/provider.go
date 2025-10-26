package kubernetes

import (
	"context"
	"strings"

	"k8s.io/client-go/kubernetes"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uckube"
	"userclouds.com/infra/uclog"
)

const (
	Prefix = "kube://secrets/"
	// TODO: Make this configurable.
	DefaultNamespace = "userclouds"
)

type Provider struct {
	client kubernetes.Interface
}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) WithClient(client kubernetes.Interface) *Provider {
	p.client = client
	return p
}

func (p *Provider) Prefix() string {
	return Prefix
}

func (p *Provider) IsDev() bool {
	return false
}

func (p *Provider) Get(ctx context.Context, path string) (string, error) {
	if err := p.initClient(); err != nil {
		return "", ucerr.Wrap(err)
	}

	secretPath := pathToSecretName(path)
	uclog.Debugf(ctx, "Getting secret %s", secretPath)
	secret, err := uckube.GetSecret(ctx, p.client, secretPath, DefaultNamespace)
	return secret, ucerr.Wrap(err)
}

func (p *Provider) Save(ctx context.Context, path, secret string) error {
	if err := p.initClient(); err != nil {
		return ucerr.Wrap(err)
	}

	err := uckube.CreateOrUpdateSecret(ctx, p.client, pathToSecretName(path), DefaultNamespace, secret)
	return ucerr.Wrap(err)
}

func (p *Provider) Delete(ctx context.Context, path string) error {
	if err := p.initClient(); err != nil {
		return ucerr.Wrap(err)
	}

	err := uckube.DeleteSecret(ctx, p.client, pathToSecretName(path), DefaultNamespace)
	return ucerr.Wrap(err)
}

func (p *Provider) initClient() error {
	if p.client != nil {
		return nil
	}

	client, err := uckube.NewClient()
	if err != nil {
		return ucerr.Wrap(err)
	}

	p.client = client
	return nil
}

// pathToSecretName turns a <service>/<name> userclouds secret path
// to a k8s compatible name.
func pathToSecretName(path string) string {
	n := strings.Replace(path, "_", "-", -1)
	return strings.Replace(n, "/", ".", -1)
}
