package nxr

import (
	"context"
	"errors"

	"github.com/datadrivers/go-nexus-client/nexus3/schema/security"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/mitchellh/mapstructure"
)

const (
	nxrUserType = "nexus_repository_user"
)

// nxrUserSecret defines a secret to store for a given role
// and how it should be revoked or renewed.
func nxrUserSecret(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: nxrUserType,
		Fields: map[string]*framework.FieldSchema{
			nxrUserType: {
				Type:        framework.TypeString,
				Description: "Nexus Repository User info",
			},
		},
		Revoke: b.nxrUserSecretRevoke,
		Renew:  b.nxrUserSecretRenew,
	}
}

// tokenRevoke removes the token from the Vault storage API and calls the client to revoke the robot account
func (b *backend) nxrUserSecretRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return logical.ErrorResponse(`error getting Nexus Repository client`), nil
	}

	userIdRaw, ok := req.Secret.InternalData["user_id"]
	if !ok {
		return logical.ErrorResponse(`"user_id" is missing on the lease`), nil
	}

	userId, ok := userIdRaw.(string)
	if !ok {
		return logical.ErrorResponse(`unable convert "user_id" to string`), nil
	}

	if err := client.deleteUser(userId); err != nil {
		return logical.ErrorResponse(`error revoking Nexus Repository user "%s"`, userId), err
	}

	return nil, nil
}

// Renew lease
func (b *backend) nxrUserSecretRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleRaw, ok := req.Secret.InternalData["role"]
	if !ok {
		return logical.ErrorResponse("secret is missing role internal data"), nil
	}

	// get the role entry
	role := roleRaw.(string)
	roleEntry, err := getRole(ctx, req.Storage, role)
	if err != nil {
		return nil, err
	}

	if roleEntry == nil {
		return nil, errors.New("error retrieving role: role is nil")
	}

	resp := &logical.Response{Secret: req.Secret}

	if roleEntry.TTL > 0 {
		resp.Secret.TTL = roleEntry.TTL
	}
	if roleEntry.MaxTTL > 0 {
		resp.Secret.MaxTTL = roleEntry.MaxTTL
	}

	return resp, nil
}

type nxrUser struct {
	UserID     string   `json:"user_id" mapstructure:"user_id"`
	Password   string   `json:"password" mapstructure:"password"`
	Email      string   `json:"email_address" mapstructure:"email_address"`
	NexusRoles []string `json:"nexus_roles" mapstructure:"nexus_roles"`
}

func (u *nxrUser) toResponseData() (map[string]interface{}, error) {
	respData := map[string]interface{}{}
	if err := mapstructure.Decode(u, &respData); err != nil {
		return nil, err
	}

	return respData, nil
}

func createNxrUser(c *nxrClient, u *nxrUser) error {
	userCreateRequest := security.User{
		UserID:       u.UserID,
		FirstName:    u.UserID,
		LastName:     u.UserID,
		EmailAddress: u.Email,
		Password:     u.Password,
		Roles:        u.NexusRoles,
		Status:       "active",
	}
	return c.createUser(userCreateRequest)
}
