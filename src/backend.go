package nxr

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	backendHelp = `
The Nexus Repository secrets backend provides dynamic user/password based on configured roles.
`
)

var Version = "v0.0.1"

// Factory configs and returns backend
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := newBackend()

	if err := b.Backend.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

// backend defines an object that extends the Vault backend and stores the API client
type backend struct {
	*framework.Backend
	client      *nxrClient
	configMutex sync.RWMutex
	rolesMutex  sync.RWMutex
	// version     string
}

// newBackend create a backend
func newBackend() *backend {
	b := &backend{}

	b.Backend = &framework.Backend{
		BackendType:    logical.TypeLogical,
		Help:           strings.TrimSpace(backendHelp),
		RunningVersion: Version,
		Invalidate:     b.invalidate,

		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{configAdminPath},
		},
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathConfigAdmin(b),
				pathConfigRotate(b),
				pathCreds(b),
			},
			pathRoles(b),
		),
		Secrets: []*framework.Secret{
			nxrUserSecret(b),
		},
	}

	return b
}

// invalidate clears an existing client configuration in
// the backend
func (b *backend) invalidate(ctx context.Context, key string) {
	if key == "config" {
		b.reset()
	}
}

// reset clears any client configuration for a new
// backend to be configured
func (b *backend) reset() {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()
	b.client = nil
}

// getClient locks the backend as it configures and creates
// a new client for the Nexus Repository API
func (b *backend) getClient(ctx context.Context, s logical.Storage) (*nxrClient, error) {
	b.configMutex.RLock()
	unlockFunc := b.configMutex.RUnlock

	//nolint:gocritic
	defer func() { unlockFunc() }()

	if b.client != nil {
		return b.client, nil
	}

	b.configMutex.RUnlock()
	b.configMutex.Lock()
	unlockFunc = b.configMutex.Unlock

	config, err := b.fetchAdminConfig(ctx, s)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = &adminConfig{}
	}

	b.client, err = newClient(config)
	if err != nil {
		return nil, err
	}

	return b.client, nil
}
