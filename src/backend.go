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

var Version = "0.0.1"

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
	configMutex sync.RWMutex
	rolesMutex  sync.RWMutex
	client      *nxrClient
	version     string
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
			SealWrapStorage: []string{pathConfigAdmin},
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
