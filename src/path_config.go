package nxr

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	// pathConfigHelpSynopsis summarizes the help text for the configuration
	pathConfigHelpSynopsis = `Configure the Nexus Repository admin configuration.`

	// pathConfigHelpDescription describes the help text for the configuration
	pathConfigHelpDescription = `
The Nexus Repository secret backend requires credentials for managing user.

You must create a username ("username" parameter)
and password ("password" parameter)
and specify the Nexus Repository address ("url" parameter)
for the API before using this secrets backend.

An optional "insecure" parameter will enable bypassing
the TLS connection verification with Nexus Repository
(when server using self-signed certificate).

An optional "timeout" is the maximum time (in second)
to wait before the request times out.
`
	pathConfigAdmin = "config/admin"
)

// adminConfig includes the minimum configuration
// required to instantiate a new Nexus Repository client.
type adminConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
	Insecure bool   `json:"insecure,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

// pathConfig extends the Vault API with a `/config`
// endpoint for the backend.
func pathConfig(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: pathConfigAdmin,
		Fields: map[string]*framework.FieldSchema{
			"username": {
				Type:        framework.TypeLowerCaseString,
				Description: "The username to access Nexus Repository API",
				Required:    true,
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Username",
					Sensitive: false,
				},
			},
			"password": {
				Type:        framework.TypeLowerCaseString,
				Description: "The user's password to access Nexus Repository API",
				Required:    true,
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Password",
					Sensitive: true,
				},
			},
			"url": {
				Type:        framework.TypeLowerCaseString,
				Description: "The URL for the Nexus Repository API",
				Required:    true,
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "URL",
					Sensitive: false,
				},
			},
			"insecure": {
				Type:        framework.TypeBool,
				Default:     false,
				Description: "Optional. Bypass certification verification for TLS connection with Nexus Repository API. Default to `false`",
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Insecure",
					Sensitive: false,
				},
			},
			"timeout": {
				Type:        framework.TypeInt,
				Default:     30,
				Description: "Optional. Timeout for connection with Nexus Repository API. Default to `30` (second)",
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Timeout",
					Sensitive: false,
				},
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathConfigRead,
				Summary:  "Examine the Nexus Repository admin configuration",
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathConfigWrite,
				Summary:  "Create the Nexus Repository admin configuration",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigWrite,
				Summary:  "Update (overwrite) the Nexus Repository admin configuration",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathConfigDelete,
				Summary:  "Delete the Nexus Repository admin configuration",
			},
		},
		ExistenceCheck:  b.pathConfigExistenceCheck,
		HelpSynopsis:    pathConfigHelpSynopsis,
		HelpDescription: pathConfigHelpDescription,
	}
}

// pathConfigExistenceCheck verifies if the configuration exists.
func (b *backend) pathConfigExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, fmt.Errorf("existence check failed: %w", err)
	}

	return out != nil, nil
}

// pathConfigRead reads the configuration and outputs non-sensitive information.
func (b *backend) pathConfigRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	config, err := fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return logical.ErrorResponse("admin configuration not found"), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"username": config.Username,
			"url":      config.URL,
			"insecure": config.Insecure,
			"timeout":  config.Timeout,
		},
	}, nil
}

// pathConfigWrite write (and force updates) the configuration for the backend
func (b *backend) pathConfigWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	createOperation := (req.Operation == logical.CreateOperation)

	config, err := fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		if !createOperation {
			return nil, errors.New("admin configuration not found during update operation")
		}
		config = new(adminConfig)
	}

	if username, ok := data.GetOk("username"); ok {
		config.Username = username.(string)
	} else if !ok && createOperation {
		return nil, fmt.Errorf("missing username in admin configuration")
	}

	if url, ok := data.GetOk("url"); ok {
		config.URL = url.(string)
	} else if !ok && createOperation {
		return nil, fmt.Errorf("missing url in admin configuration")
	}

	if password, ok := data.GetOk("password"); ok {
		config.Password = password.(string)
	} else if !ok && createOperation {
		return nil, fmt.Errorf("missing password in admin configuration")
	}

	if insecure, ok := data.GetOk("insecure"); ok {
		config.Insecure = insecure.(bool)
	}

	if timeout, ok := data.GetOk("timeout"); ok {
		config.Timeout = timeout.(int)
	}

	entry, err := logical.StorageEntryJSON(pathConfigAdmin, config)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	// reset the client so the next invocation will pick up the new configuration
	b.client = nil

	return nil, nil
}

// pathConfigDelete removes the configuration for the backend
func (b *backend) pathConfigDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	config, err := fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return logical.ErrorResponse("admin configuration not found"), nil
	}

	err = req.Storage.Delete(ctx, pathConfigAdmin)
	if err == nil {
		b.client = nil
	}

	return nil, err
}

func fetchAdminConfig(ctx context.Context, s logical.Storage) (*adminConfig, error) {
	entry, err := s.Get(ctx, pathConfigAdmin)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	config := new(adminConfig)
	if err := entry.DecodeJSON(&config); err != nil {
		return nil, fmt.Errorf("error fetching admin configuration: %w", err)
	}

	// return the config, we are done
	return config, nil
}
