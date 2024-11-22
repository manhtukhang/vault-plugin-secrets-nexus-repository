package nxr

import (
	"errors"

	nexus "github.com/datadrivers/go-nexus-client/nexus3"
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
)

// nxrClient creates an object storing the client.
type nxrClient struct {
	*nexus.NexusClient
}

// newClient creates a new client to access Nexus Repository
// and exposes it for any secrets or roles to use.
func newClient(config *adminConfig) (*nxrClient, error) {
	if config == nil {
		return nil, errors.New("client configuration was nil")
	}

	if config.Username == "" {
		return nil, errors.New("client username was not defined")
	}

	if config.Password == "" {
		return nil, errors.New("client password was not defined")
	}

	if config.URL == "" {
		return nil, errors.New("client URL was not defined")
	}
	c := nexus.NewClient(client.Config{
		URL:      config.URL,
		Username: config.Username,
		Password: config.Password,
		Insecure: config.Insecure,
		Timeout:  &config.Timeout,
	})

	return &nxrClient{c}, nil
}

func (c *nxrClient) createUser(userCreateRequest security.User) error {
	return c.Security.User.Create(userCreateRequest)
}

func (c *nxrClient) deleteUser(userID string) error {
	return c.Security.User.Delete(userID)
}

func (c *nxrClient) changeUserPassword(userID string, password string) error {
	return c.Security.User.ChangePassword(userID, password)
}
