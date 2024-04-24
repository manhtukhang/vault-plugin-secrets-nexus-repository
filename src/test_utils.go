package nxr

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

func writeConfigAdmin(b logical.Backend, s logical.Storage, d map[string]interface{}) (*logical.Response, error) {
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      configAdminPath,
		Storage:   s,
		Data:      d,
	})
}

func readConfigAdmin(b logical.Backend, s logical.Storage) (*logical.Response, error) {
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      configAdminPath,
		Storage:   s,
	})
}

func deleteConfigAdmin(b logical.Backend, s logical.Storage) (*logical.Response, error) {
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      configAdminPath,
		Storage:   s,
	})
}
