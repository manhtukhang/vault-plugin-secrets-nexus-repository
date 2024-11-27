package nxr

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	configAdminPath = "config/admin"
	defaultTimeout  = 30
	defaultInsecure = false
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

// pathConfigAdmin extends the Vault API with a `config/admin`
// endpoint for the backend.
func pathConfigAdmin(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: configAdminPath,
		Fields: map[string]*framework.FieldSchema{
			"username": {
				Type:        framework.TypeLowerCaseString,
				Description: "The username to access Nexus Repository API.",
				Required:    true,
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Username",
					Sensitive: false,
				},
			},
			"password": {
				Type:        framework.TypeString,
				Description: "The user's password to access Nexus Repository API.",
				Required:    true,
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Password",
					Sensitive: true,
				},
			},
			"url": {
				Type:        framework.TypeLowerCaseString,
				Description: "The URL for the Nexus Repository API.",
				Required:    true,
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "URL",
					Sensitive: false,
				},
			},
			"insecure": {
				Type:        framework.TypeBool,
				Default:     defaultInsecure,
				Description: "Optional. Bypass certification verification for TLS connection with Nexus Repository API. Default to `false`.",
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Insecure",
					Sensitive: false,
				},
			},
			"timeout": {
				Type:        framework.TypeDurationSecond,
				Default:     defaultTimeout,
				Description: "Optional. Timeout for connection with Nexus Repository API. Default to `30s` (30 seconds).",
				DisplayAttrs: &framework.DisplayAttributes{
					Name:      "Timeout",
					Sensitive: false,
				},
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathConfigAdminRead,
				Summary:  "Examine the Nexus Repository admin configuration.",
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathConfigAdminWrite,
				Summary:  "Create the Nexus Repository admin configuration.",
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigAdminWrite,
				Summary:  "Update (overwrite) the Nexus Repository admin configuration.",
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathConfigAdminDelete,
				Summary:  "Delete the Nexus Repository admin configuration.",
			},
		},
		ExistenceCheck:  b.pathExistenceCheck,
		HelpSynopsis:    pathConfigAdminHelpSynopsis,
		HelpDescription: pathConfigAdminHelpDescription,
	}
}

// pathExistenceCheck verifies if the configuration of path exists.
func (b *backend) pathExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, err
	}

	return out != nil, nil
}

// pathConfigAdminRead reads the configuration and outputs non-sensitive information.
func (b *backend) pathConfigAdminRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	config, err := b.fetchAdminConfig(ctx, req.Storage)
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

// pathConfigAdminWrite write (and force updates) the configuration for the backend
func (b *backend) pathConfigAdminWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	config, err := b.fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = &adminConfig{}
	}

	createOperation := (req.Operation == logical.CreateOperation)

	if username, ok := data.GetOk("username"); ok {
		config.Username = username.(string)
	}

	if url, ok := data.GetOk("url"); ok {
		config.URL = url.(string)
		config.Password = "" // NOTE: clear password if URL changes, requires setting password and url together for security reasons
	}

	if password, ok := data.GetOk("password"); ok {
		config.Password = password.(string)
	}

	if insecure, ok := data.GetOk("insecure"); ok {
		config.Insecure = insecure.(bool)
	} else if createOperation {
		config.Insecure = data.Get("insecure").(bool)
	}

	if timeout, ok := data.GetOk("timeout"); ok {
		config.Timeout = timeout.(int)
	} else if createOperation {
		config.Timeout = data.Get("timeout").(int)
	}

	// Verify
	if config.Username == "" {
		return logical.ErrorResponse(`missing "username" in admin configuration`), nil
	}

	if config.URL == "" {
		return logical.ErrorResponse(`missing "url" in admin configuration`), nil
	}

	if config.Password == "" {
		return logical.ErrorResponse(`missing "password" in admin configuration`), nil
	}

	entry, err := logical.StorageEntryJSON(configAdminPath, config)
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

// pathConfigAdminDelete removes the configuration for the backend
func (b *backend) pathConfigAdminDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.configMutex.Lock()
	defer b.configMutex.Unlock()

	config, err := b.fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return logical.ErrorResponse("admin configuration not found"), nil
	}

	err = req.Storage.Delete(ctx, configAdminPath)
	if err == nil {
		b.client = nil
	}

	return nil, err
}

// fetchAdminConfig fetches admin configuration for the backend
func (b *backend) fetchAdminConfig(ctx context.Context, s logical.Storage) (*adminConfig, error) {
	entry, err := s.Get(ctx, configAdminPath)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	config := &adminConfig{}
	if err := entry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	// return the config, we are done
	return config, nil
}

const (
	pathConfigAdminHelpSynopsis = `Configure the Nexus Repository admin configuration.`

	pathConfigAdminHelpDescription = `
The Nexus Repository secret backend requires credentials for managing user.

You must create a username ("username" parameter)
and password ("password" parameter)
and specify the Nexus Repository address ("url" parameter)
for the API before using this secrets backend.

An optional "insecure" parameter will enable bypassing
the TLS connection verification with Nexus Repository
(when server using self-signed certificate).

An optional "timeout" parameter is the maximum time (in seconds)
to wait before the request to the API is timed out.
`
)
