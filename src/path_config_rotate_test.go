package nxr

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"

	"go.nhat.io/httpmock"
)

const (
	userChangePasswordURI = "/service/rest/v1/security/users/%s/change-password"
)

func TestConfigRotateWithMockApi(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	t.Run("Unhappy cases", func(t *testing.T) {
		// Rotate empty config
		err := testConfigRotateWrite(b, reqStorage)
		assert.Error(t, err)

		// GatewayTimeout
		srv := httpmock.New(func(s *httpmock.Server) {
			s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
				ReturnCode(httpmock.StatusGatewayTimeout)
		})(t)

		_, err = writeConfigAdmin(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      srv.URL(),
		})
		assert.NoError(t, err)

		err = testConfigRotateWrite(b, reqStorage)
		assert.Error(t, err)

		// User does not have permission to perform the operation
		srv = httpmock.New(func(s *httpmock.Server) {
			s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
				ReturnCode(httpmock.StatusForbidden)
		})(t)

		_, err = writeConfigAdmin(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      srv.URL(),
		})
		assert.NoError(t, err)

		err = testConfigRotateWrite(b, reqStorage)
		assert.Error(t, err)
	})

	t.Run("Happy cases", func(t *testing.T) {
		srv := httpmock.New(func(s *httpmock.Server) {
			s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
				ReturnCode(httpmock.StatusOK)
		})(t)

		_, err := writeConfigAdmin(b, reqStorage, map[string]interface{}{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      srv.URL(),
		})
		assert.NoError(t, err)

		err = testConfigRotateWrite(b, reqStorage)
		assert.NoError(t, err)
	})
}

func testConfigRotateWrite(b logical.Backend, s logical.Storage) error {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      configRotatePath,
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
