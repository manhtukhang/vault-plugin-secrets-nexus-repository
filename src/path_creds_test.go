package nxr

import (
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.nhat.io/httpmock"
)

const (
	userCreateURI = "/service/rest/v1/security/users"
	userURI       = "/service/rest/v1/security/users/%s"
	testCredsPath = credsPath + testRoleName
)

func Test_Creads(t *testing.T) {
	t.Run("Creds_Fail", test_Creds_Fail)
	t.Run("Creds_WithMockApi", testCreds_WithMockApi)
	t.Run("Creds_WithMockApi_Fail", testCreds_WithMockApi_Fail)
}

func test_Creds_Fail(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	// Role does not exist
	expectedError := fmt.Sprintf(`role "%s" does not exist`, testRoleName)
	resp, err := doAction(actionRead, testCredsPath, b, reqStorage, nil)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.IsError())
	assert.Equal(t, expectedError, resp.Error().Error())

	// Unsuported operations
	expectedError = `unsupported operation`
	for _, v := range []logical.Operation{actionCreate, actionUpdate, actionDelete, actionList} {
		resp, err = doAction(v, testCredsPath, b, reqStorage, nil)

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedError, err.Error())

	}
}

func testCreds_WithMockApi(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	mockSrv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectPost(userCreateURI).
			ReturnCode(httpmock.StatusOK)
	})(t)

	config := &testData{
		"username": testConfigAdminUsername,
		"password": testConfigAdminPassword,
		"url":      mockSrv.URL(),
	}
	// Create base config
	resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, *config)
	require.NoError(t, err)
	assert.Nil(t, resp)

	// Create role
	roleData := testData{
		"name":             testRoleName,
		"nexus_roles":      testRoleNexusRoles,
		"user_id_template": testRoleUserIdTemplate,
		"user_email":       testRoleUserEmail,
		"ttl":              10,
		"max_ttl":          30,
	}
	resp, err = doAction(actionCreate, rolesPath+testRoleName, b, reqStorage, roleData)
	require.NoError(t, err)
	assert.Nil(t, resp)

	// test get cred (Nexus user)
	resp, err = doAction(actionRead, testCredsPath, b, reqStorage, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func testCreds_WithMockApi_Fail(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	mockSrv := httpmock.New(func(s *httpmock.Server) {
		s.ExpectPost(userCreateURI).
			ReturnCode(httpmock.StatusBadGateway)
	})(t)

	config := &testData{
		"username": testConfigAdminUsername,
		"password": testConfigAdminPassword,
		"url":      mockSrv.URL(),
	}
	// Create base config
	resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, *config)
	require.NoError(t, err)
	assert.Nil(t, resp)

	// Create role
	roleData := testData{
		"name":             testRoleName,
		"nexus_roles":      testRoleNexusRoles,
		"user_id_template": testRoleUserIdTemplate,
		"user_email":       testRoleUserEmail,
		"ttl":              10,
		"max_ttl":          30,
	}
	resp, err = doAction(actionCreate, rolesPath+testRoleName, b, reqStorage, roleData)
	require.NoError(t, err)
	assert.Nil(t, resp)

	// test get cred (Nexus user), expect error
	resp, err = doAction(actionRead, testCredsPath, b, reqStorage, nil)
	require.Error(t, err)
	assert.Nil(t, resp)
}
