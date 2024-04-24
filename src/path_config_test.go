package nxr

import (
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

const (
	testConfigAdminUsername = "admin"
	testConfigAdminPassword = "Testing!123"
	testConfigAdminURL      = "http://localhost:1234"
	testConfigAdminInsecure = true
	testConfigAdminTimeout  = 30
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
			"password": testConfigAdminPassword,
			"url":      testConfigAdminURL,
		})
		assert.Error(t, err)

		// Missing "password"
		err = testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"url":      testConfigAdminURL,
		})
		assert.Error(t, err)

		// Missing "url"
		err = testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
		})
		assert.Error(t, err)
	})

	t.Run("Happy cases", func(t *testing.T) {
		// No "insecure" + "timeout"
		err := testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      testConfigAdminURL,
		})
		assert.NoError(t, err)

		// "insecure"
		err = testConfigCreate(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      testConfigAdminURL,
			"insecure": testConfigAdminInsecure,
		})
		assert.NoError(t, err)

		// Update after create
		err = testConfigUpdate(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      testConfigAdminURL,
			"insecure": !testConfigAdminInsecure,
			"timeout":  testConfigAdminTimeout,
		})
		assert.NoError(t, err)

		// Read after update
		err = testConfigRead(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"url":      testConfigAdminURL,
			"insecure": !testConfigAdminInsecure,
			"timeout":  testConfigAdminTimeout,
		})
		assert.NoError(t, err)

		// Delete
		err = testConfigDelete(b, reqStorage)
		assert.NoError(t, err)
	})
}

func testConfigDelete(b logical.Backend, s logical.Storage) error {
	resp, err := deleteConfigAdmin(b, s)
	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testConfigCreate(b logical.Backend, s logical.Storage, d map[string]interface{}) error {
	resp, err := writeConfigAdmin(b, s, d)
	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testConfigUpdate(b logical.Backend, s logical.Storage, d map[string]interface{}) error {
	resp, err := writeConfigAdmin(b, s, d)
	if err != nil {
		return err
	}

	if resp != nil && resp.IsError() {
		return resp.Error()
	}
	return nil
}

func testConfigRead(b logical.Backend, s logical.Storage, expected map[string]interface{}) error {
	resp, err := readConfigAdmin(b, s)
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
