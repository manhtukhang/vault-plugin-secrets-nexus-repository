package nxr

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.nhat.io/httpmock"
)

const (
	userChangePasswordURI = "/service/rest/v1/security/users/%s/change-password"
)

func Test_ConfigRotate(t *testing.T) {
	t.Run("ConfigRotate_Fail", TestConfigRotate_Fail)
	t.Run("ConfigRotate_WithMockApi", TestConfigRotate_WithMockApi)
	t.Run("ConfigRotate_WithMockApi_Fail", TestConfigRotate_WithMockApi_Fail)
}

func TestConfigRotate_Fail(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	// Rotate empty config
	expectedError := `admin configuration not found`

	resp, err := doAction(actionUpdate, configRotatePath, b, reqStorage, nil)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsError())
	assert.Equal(t, expectedError, resp.Error().Error())

	// Unsupported operations
	expectedError = `unsupported operation`
	for _, v := range []logical.Operation{actionCreate, actionRead, actionDelete} {
		resp, err = doAction(v, configRotatePath, b, reqStorage, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedError, err.Error())

	}
}

func TestConfigRotate_WithMockApi(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	mockSrv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
			ReturnCode(httpmock.StatusOK)
	})(t)

	data := &testData{
		"username": testConfigAdminUsername,
		"password": testConfigAdminPassword,
		"url":      mockSrv.URL(),
	}

	// Create base config
	resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, *data)

	require.NoError(t, err)
	assert.Nil(t, resp)

	// Rotate
	resp, err = doAction(actionUpdate, configRotatePath, b, reqStorage, nil)

	require.NoError(t, err)
	assert.Nil(t, resp)
}

func TestConfigRotate_WithMockApi_Fail(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	testCases := []struct {
		mockSrv       *httpmock.Server
		expectedError string
	}{
		{
			mockSrv: httpmock.New(func(s *httpmock.Server) {
				s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
					ReturnCode(httpmock.StatusGatewayTimeout)
			})(t),
			expectedError: fmt.Sprintf(`could not change password of user 'admin':  HTTP: %d`, httpmock.StatusGatewayTimeout),
		},
		{
			mockSrv: httpmock.New(func(s *httpmock.Server) {
				s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
					ReturnCode(httpmock.StatusBadGateway)
			})(t),
			expectedError: fmt.Sprintf(`could not change password of user 'admin':  HTTP: %d`, httpmock.StatusBadGateway),
		},
		{
			mockSrv: httpmock.New(func(s *httpmock.Server) {
				s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
					ReturnCode(httpmock.StatusRequestTimeout)
			})(t),
			expectedError: fmt.Sprintf(`could not change password of user 'admin':  HTTP: %d`, httpmock.StatusRequestTimeout),
		},
		{
			mockSrv: httpmock.New(func(s *httpmock.Server) {
				s.ExpectPut(fmt.Sprintf(userChangePasswordURI, testConfigAdminUsername)).
					ReturnCode(httpmock.StatusOK).
					After(2 * time.Second)
			})(t),
			expectedError: `could not change password of user 'admin':.*(Client.Timeout exceeded while awaiting headers)`,
		},
	}

	for _, tc := range testCases {
		_, err := doAction(actionCreate, configAdminPath, b, reqStorage, testData{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      tc.mockSrv.URL(),
			"timeout":  "1s",
		})
		require.NoError(t, err)

		resp, err := doAction(actionUpdate, configRotatePath, b, reqStorage, nil)
		assert.Error(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
		assert.Nil(t, resp)
	}
}
