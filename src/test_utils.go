package nxr

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

const (
	actionCreate = logical.CreateOperation
	actionUpdate = logical.UpdateOperation
	actionRead   = logical.ReadOperation
	actionDelete = logical.DeleteOperation
	actionList   = logical.ListOperation
	actionRevoke = logical.RevokeOperation
	actionRenew  = logical.RenewOperation
)

type testData map[string]interface{}

func doAction(action logical.Operation, p string, b logical.Backend, s logical.Storage, d testData) (*logical.Response, error) {
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: action,
		Path:      p,
		Storage:   s,
		Data:      d,
	})
}

func doSecretAction(action logical.Operation, r *logical.Secret, b logical.Backend, s logical.Storage) (*logical.Response, error) {
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: action,
		Secret:    r,
		Storage:   s,
	})
}
