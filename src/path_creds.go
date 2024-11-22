package nxr

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/template"
	"github.com/hashicorp/vault/sdk/logical"
	gopw "github.com/sethvargo/go-password/password"
)

const (
	credsPath = "creds/"
)

// UserIdMetadata defines the metadata that a user_id_template can use
// to dynamically create user account in Nexus Repository
type userIdMetadata struct {
	DisplayName string
	RoleName    string
}

func pathCreds(b *backend) *framework.Path {
	forwardOperation := &framework.PathOperation{
		Callback:                    b.pathCredentialsRead,
		ForwardPerformanceSecondary: true,
		ForwardPerformanceStandby:   true,
	}
	return &framework.Path{
		Pattern: credsPath + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeLowerCaseString,
				Description: "Name of the role.",
				Required:    true,
			},
			// "ttl": {
			// 	Type:        framework.TypeDurationSecond,
			// 	Description: "Optional. Default lease for generated users. If not set or set to 0, will use system default.",
			// },
		},

		HelpSynopsis:    pathCredsHelpSyn,
		HelpDescription: pathCredsHelpDesc,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: forwardOperation,
		},
	}
}

func (b *backend) pathCredentialsRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name").(string)

	roleEntry, err := getRole(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}

	if roleEntry == nil {
		return logical.ErrorResponse(fmt.Sprintf(`role "%s" does not exist`, roleName)), nil
	}

	return b.creadCred(ctx, req, roleEntry)
}

func (b *backend) creadCred(ctx context.Context, req *logical.Request, role *nxrRoleEntry) (*logical.Response, error) {
	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	var displayName string
	if req.DisplayName != "" {
		re := regexp.MustCompile("[^[:alnum:]._-]")
		displayName = re.ReplaceAllString(req.DisplayName, "-")
	}

	up, _ := template.NewTemplate(template.Template(role.UserIdTemplate)) // this was verified in role config
	generatedUserId, err := up.Generate(userIdMetadata{
		DisplayName: displayName,
		RoleName:    role.Name,
	})
	if err != nil {
		return nil, err
	}

	randomPassword, err := gopw.Generate(64, 10, 10, false, false)
	if err != nil {
		return nil, err
	}

	userReq := &nxrUser{
		UserID:     generatedUserId,
		Password:   randomPassword,
		Email:      role.UserEmail,
		NexusRoles: role.NexusRoles,
	}

	err = createNxrUser(client, userReq)
	if err != nil {
		return nil, err
	}

	responseData, err := userReq.toResponseData()
	if err != nil {
		return nil, err
	}

	internalData := map[string]interface{}{
		"role":    role.Name,
		"user_id": generatedUserId,
	}

	resp := b.Secret(nxrUserType).Response(responseData, internalData)

	if role.TTL > 0 {
		resp.Secret.TTL = role.TTL
	}

	if role.MaxTTL > 0 {
		resp.Secret.MaxTTL = role.MaxTTL
	}

	return resp, nil
}

const (
	pathCredsHelpSyn  = `Request Nexus Repository user credentials for a given Vault role.`
	pathCredsHelpDesc = `
This path creates dynamic Nexus Repository user credentials.
The associated Vault role can be configured to create a new
user bound to a list of existing Nexus security roles.
The user created in Nexus Repository server will be
automatically deleted when the lease has expired.
`
)
