package nxr

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/template"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/mitchellh/mapstructure"
)

const (
	rolesPath                  = "roles/"
	defaultUserIdTemplate      = `{{ printf "v-%s-%s-%s-%s" (.RoleName | truncate 64) (.DisplayName | truncate 64) (unix_time) (random 24) | truncate 192 | lowercase }}`
	defaultUserEmail           = "no-one@example.org" // Suppose that the email domain will never be owned by any organization or individual
	emailValidationRegexString = "^(?:(?:(?:(?:[a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(?:\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|(?:(?:\\x22)(?:(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(?:\\x20|\\x09)+)?(?:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(\\x20|\\x09)+)?(?:\\x22))))@(?:(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$"
)

var emailValidationRegex = regexp.MustCompile(emailValidationRegexString)

// nxrRoleEntry defines the data required for a Vault role
// to access and call the Nexus Repository API endpoints
type nxrRoleEntry struct {
	Name           string        `json:"name" mapstructure:"name"`
	NexusRoles     []string      `json:"nexus_roles" mapstructure:"nexus_roles"`
	UserIdTemplate string        `json:"user_id_template" mapstructure:"user_id_template"`
	UserEmail      string        `json:"user_email" mapstructure:"user_email"`
	TTL            time.Duration `json:"ttl" mapstructure:"ttl"`
	MaxTTL         time.Duration `json:"max_ttl" mapstructure:"max_ttl"`
	// NexusRolesCheck bool          `json:"nexus_roles_check" mapstructure:"nexus_roles_check"`
	// Cache           bool          `json:"cache" mapstructure:"cache"`
}

// toResponseData returns response data for a role
func (r *nxrRoleEntry) toResponseData() (map[string]interface{}, error) {
	respData := map[string]interface{}{}

	err := mapstructure.Decode(r, &respData)
	if err != nil {
		return nil, err
	}

	// Using seconds as format for TTLs
	respData["ttl"] = r.TTL.Seconds()
	respData["max_ttl"] = r.MaxTTL.Seconds()

	return respData, err
}

// pathRoles extends the Vault API with a `/roles`
// endpoint for the backend.
func pathRoles(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: rolesPath + framework.GenericNameRegex("name"),
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeNameString,
					Description: "Name of the role.",
					Required:    true,
				},
				"nexus_roles": {
					Type:        framework.TypeCommaStringSlice,
					Description: "The Nexus Repository roles for the user.",
					Required:    true,
				},
				"user_id_template": {
					Type:        framework.TypeString,
					Description: fmt.Sprintf("Optional. Template to generate UserId field for the user. Default to %s.", defaultUserIdTemplate),
					Default:     defaultUserIdTemplate,
				},
				"user_email": {
					Type:        framework.TypeString,
					Description: fmt.Sprintf("Optional. Email field for the user. Default to %s.", defaultUserEmail),
					Default:     defaultUserEmail,
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Optional. Default lease for generated users. If not set or set to 0, will use system default.",
				},
				"max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Optional. Maximum lease time for generated users. If not set or set to 0, will use system default.",
				},
				// TODO: check if all nexus_roles are existing on Nexus Repository server to allow create the role
				// "nexus_roles_check": {
				// 	Type:        framework.TypeBool,
				// 	Description: "Optional. Check if all nexus_roles are existing on Nexus Repository server before create the role. If not set or set to false, will skip the checking.",
				// 	Default:     false,
				// },
				// TODO: cache and response the previous created user for next requests (within max_ttl) to reduce API abusing
				// "cache": {
				// 	Type:        framework.TypeBool,
				// 	Description: "Optional. Cache the previous created user in this role (from a same bound claim user) to avoid creating to many users with the same privileges. Default to false.",
				// 	Default:     false,
				// },
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathRolesRead,
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathRolesWrite,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathRolesWrite,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathRolesDelete,
				},
			},
			HelpSynopsis:    pathRolesHelpSynopsis,
			HelpDescription: pathRolesHelpDescription,
			ExistenceCheck:  b.pathRolesExistenceCheck,
		},
		{
			Pattern: rolesPath + "?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathRolesList,
				},
			},
			HelpSynopsis:    pathRolesListHelpSynopsis,
			HelpDescription: pathRolesListHelpDescription,
		},
	}
}

// pathRolesExistenceCheck verifies if the role exists
func (b *backend) pathRolesExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, err
	}

	return out != nil, nil
}

// pathRolesList makes a request to Vault storage to retrieve a list of roles for the backend
func (b *backend) pathRolesList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.rolesMutex.RLock()
	defer b.rolesMutex.RUnlock()

	entries, err := req.Storage.List(ctx, rolesPath)
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}

// pathRolesRead makes a request to Vault storage to read a role and return response data
func (b *backend) pathRolesRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.rolesMutex.RLock()
	b.configMutex.RLock()
	defer b.configMutex.RUnlock()
	defer b.rolesMutex.RUnlock()

	config, err := b.fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return logical.ErrorResponse("admin configuration not found"), nil
	}

	entry, err := getRole(ctx, req.Storage, d.Get("name").(string))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	respData, err := entry.toResponseData()
	if err != nil {
		return nil, err
	}
	return &logical.Response{Data: respData}, nil
}

// pathRolesWrite makes a request to Vault storage to update a role
// based on the attributes are passed to the role configuration
func (b *backend) pathRolesWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.rolesMutex.RLock()
	b.configMutex.RLock()
	defer b.configMutex.RUnlock()
	defer b.rolesMutex.RUnlock()

	config, err := b.fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return logical.ErrorResponse("admin configuration not found"), nil
	}

	name := d.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing role name"), nil
	}

	entry, err := getRole(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		entry = &nxrRoleEntry{
			Name: name,
		}
	}

	createOperation := (req.Operation == logical.CreateOperation)

	if nexusRolesRaw, ok := d.GetOk("nexus_roles"); ok {
		entry.NexusRoles = nexusRolesRaw.([]string)
	} else if !ok && createOperation {
		return logical.ErrorResponse(`missing "nexus_roles" in role definition`), nil
	}

	entry.UserIdTemplate = d.Get("user_id_template").(string)

	entry.UserEmail = d.Get("user_email").(string)

	if ttlRaw, ok := d.GetOk("ttl"); ok {
		entry.TTL = time.Duration(ttlRaw.(int)) * time.Second
	} else if createOperation {
		entry.TTL = time.Duration(d.Get("ttl").(int)) * time.Second
	}

	if maxTTLRaw, ok := d.GetOk("max_ttl"); ok {
		entry.MaxTTL = time.Duration(maxTTLRaw.(int)) * time.Second
	} else if createOperation {
		entry.MaxTTL = time.Duration(d.Get("max_ttl").(int)) * time.Second
	}

	// Verifying
	if _, err := template.NewTemplate(template.Template(entry.UserIdTemplate)); err != nil {
		return logical.ErrorResponse(`unable to initialize "user_id_template"`), err
	}

	if !emailValidationRegex.MatchString(entry.UserEmail) {
		return logical.ErrorResponse(`"user_email" is not valid`), nil
	}

	if entry.MaxTTL != 0 && entry.TTL > entry.MaxTTL {
		return logical.ErrorResponse(`"ttl" cannot be greater than "max_ttl"`), nil
	}

	if err := setRole(ctx, req.Storage, name, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

// pathRolesDelete makes a request to Vault storage to delete a role
func (b *backend) pathRolesDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.rolesMutex.RLock()
	b.configMutex.RLock()
	defer b.configMutex.RUnlock()
	defer b.rolesMutex.RUnlock()

	config, err := b.fetchAdminConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return logical.ErrorResponse("admin configuration not found"), nil
	}

	err = req.Storage.Delete(ctx, rolesPath+d.Get("name").(string))
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// setRole adds the role to the Vault storage API
func setRole(ctx context.Context, s logical.Storage, name string, roleEntry *nxrRoleEntry) error {
	entry, err := logical.StorageEntryJSON(rolesPath+name, roleEntry)
	if err != nil {
		return err
	}

	if entry == nil {
		return fmt.Errorf("failed to create storage entry for role")
	}

	if err := s.Put(ctx, entry); err != nil {
		return err
	}

	return nil
}

// getRole gets the role from the Vault storage API
func getRole(ctx context.Context, s logical.Storage, name string) (*nxrRoleEntry, error) {
	if name == "" {
		return nil, fmt.Errorf("missing role name")
	}

	entry, err := s.Get(ctx, rolesPath+name)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	var role nxrRoleEntry

	if err := entry.DecodeJSON(&role); err != nil {
		return nil, err
	}
	return &role, nil
}

const (
	pathRolesHelpSynopsis        = `Manage the roles that can be created with this secrets engine.`
	pathRolesHelpDescription     = `This path lets you manage the roles that can be created with this secrets engine.`
	pathRolesListHelpSynopsis    = `List the existing roles in this secrets engine.`
	pathRolesListHelpDescription = `A list of existing role names will be returned.`
)
