package nxr

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	gopw "github.com/sethvargo/go-password/password"
)

const (
	configRotatePath = "config/rotate"
)

// pathConfigRotate replaces the configurated admin's password
// with a random one.
func pathConfigRotate(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: configRotatePath,
		// TODO: rotate by creating a new user with same privileges and revoking the current one
		// Fields: map[string]*framework.FieldSchema{
		// 	"username": {
		// 		Type:        framework.TypeLowerCaseString,
		// 		Description: "Optional. Overwrite the username to access Nexus Repository API",
		// 		DisplayAttrs: &framework.DisplayAttributes{
		// 			Name:      "Username",
		// 			Sensitive: false,
		// 		},
		// 	},
		// },
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigRotateWrite,
				Summary:  "Rotate the Nexus Repository admin credential",
			},
		},
		HelpSynopsis:    pathConfigRotateHelpSynopsis,
		HelpDescription: pathConfigRotateHelpDescription,
	}
}

func (b *backend) pathConfigRotateWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config, err := b.fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return logical.ErrorResponse("admin configuration not found"), nil
	}

	// TODO: allow user configs password complexity
	newPw, err := gopw.Generate(64, 10, 0, false, true)
	if err != nil {
		return nil, err
	}

	nxrClient, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if err = nxrClient.changeUserPassword(config.Username, newPw); err != nil {
		return nil, err
	}

	// TODO: check if new password is usable (assume to yes)

	config.Password = newPw

	entry, err := logical.StorageEntryJSON(configAdminPath, config)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	// reset the client so the next invocation will pick up the new configuration
	b.reset()

	return nil, nil
}

const (
	pathConfigRotateHelpSynopsis = `Rotate the Nexus Repository admin credential.`

	pathConfigRotateHelpDescription = `
This will rotate the "password" used to access Nexus Repository from this plugin.
`
)
