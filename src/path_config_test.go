package nxr

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

const (
	username = "admin"
	password = "Testing!123"
	url      = "http://localhost:1234"
	insecure = true
	timeout  = 30
)

// TestConfig mocks the creation, read, update, and delete
// of the backend configuration
func TestConfig(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	t.Run("Unhappy cases", func(t *testing.T) {
		// Read empty config
		err := testConfigRead(b, reqStorage, nil)
		assert.Error(t, err)

		// Delete empty config
		err = testConfigDelete(b, reqStorage)
		assert.Error(t, err)

		// Missing "username"
		err = testConfigCreate(b, reqStorage, map[string]interface{}{
			"password": password,
			"url":      url,
		})
		assert.Error(t, err)

		// Missing "password"
		err = testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": username,
			"url":      url,
		})
		assert.Error(t, err)

		// Missing "url"
		err = testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": username,
			"password": password,
		})
		assert.Error(t, err)
	})

	t.Run("Happy cases", func(t *testing.T) {
		// No "insecure" + "timeout"
		err := testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": username,
			"password": password,
			"url":      url,
		})
		assert.NoError(t, err)

		// "insecure"
		err = testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": username,
			"password": password,
			"url":      url,
			"insecure": insecure,
		})
		assert.NoError(t, err)

		// Update after create
		err = testConfigUpdate(b, reqStorage, map[string]interface{}{
			"username": username,
			"password": password,
			"url":      url,
			"insecure": !insecure,
			"timeout":  timeout,
		})
		assert.NoError(t, err)

		// Read after update
		err = testConfigRead(b, reqStorage, map[string]interface{}{
			"username": username,
			"url":      url,
			"insecure": !insecure,
			"timeout":  timeout,
		})
		assert.NoError(t, err)

		// Delete
		err = testConfigDelete(b, reqStorage)
		assert.NoError(t, err)
	})
}

func testConfigDelete(b logical.Backend, s logical.Storage) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      configAdminPath,
		Storage:   s,
	})
	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testConfigCreate(b logical.Backend, s logical.Storage, d map[string]interface{}) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      configAdminPath,
		Data:      d,
		Storage:   s,
	})
	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testConfigUpdate(b logical.Backend, s logical.Storage, d map[string]interface{}) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      configAdminPath,
		Data:      d,
		Storage:   s,
	})
	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testConfigRead(b logical.Backend, s logical.Storage, expected map[string]interface{}) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      configAdminPath,
		Storage:   s,
	})
	if err != nil {
		return err
	}

	if resp == nil && expected == nil {
		return nil
	}

	if resp.IsError() {
		return resp.Error()
	}

	if len(expected) != len(resp.Data) {
		return fmt.Errorf("read data mismatch (expected %d values, got %d)", len(expected), len(resp.Data))
	}

	for k, expectedV := range expected {
		actualV, ok := resp.Data[k]

		if !ok {
			return fmt.Errorf(`expected data["%s"] = %v but was not included in read output"`, k, expectedV)
		} else if expectedV != actualV {
			return fmt.Errorf(`expected data["%s"] = %v, instead got %v"`, k, expectedV, actualV)
		}
	}

	return nil
}
