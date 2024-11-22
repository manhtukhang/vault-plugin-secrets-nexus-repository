package nxr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.nhat.io/httpmock"
)

func Test_Secrets(t *testing.T) {
	t.Run("Secret_WithMockApi", testScret_WithMockApi)
	t.Run("Secret_WithMockApi_Fail", testScret_WithMockApi_Fail)
}

func testScret_WithMockApi(t *testing.T) {
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

	// Run test get cred (Nexus user)
	resp, err = doAction(actionRead, testCredsPath, b, reqStorage, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Run test renew
	resp, err = doSecretAction(actionRenew, resp.Secret, b, reqStorage)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	userID := resp.Secret.InternalData["user_id"].(string)
	mockSrv.ExpectDelete(fmt.Sprintf(userURI, userID)) // update client URI
	// Run test revoke
	resp, err = doSecretAction(actionRevoke, resp.Secret, b, reqStorage)
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func testScret_WithMockApi_Fail(t *testing.T) {
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

	// Run test get cred (Nexus user)
	resp, err = doAction(actionRead, testCredsPath, b, reqStorage, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	//
	userID := resp.Secret.InternalData["user_id"].(string)
	mockSrv.ExpectDelete(fmt.Sprintf(userURI, userID)). // update client URI
								ReturnCode(httpmock.StatusBadGateway)
	// Run test revoke, expect error
	resp, err = doSecretAction(actionRevoke, resp.Secret, b, reqStorage)
	require.Error(t, err)
	assert.NotNil(t, resp)
}
