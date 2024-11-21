package nxr

import (
	// "fmt"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRoleName                 = "test-role"
	testRoleNexusRoles           = "nx-test1,nx-test2"
	testRoleUserIdTemplate       = `{{ printf "v.%s.%s.%s.%s" (.RoleName | truncate 64) (.DisplayName | truncate 64) (unix_time) (random 24) | truncate 192 | lowercase }}`
	testRoleUserEmail            = "no-one@example.org"
	testRoleNameUpdate           = "test-role-update"
	testRoleNexusRolesUpdate     = "nx-test1,nx-test2,nx-test3"
	testRoleUserIdTemplateUpdate = `{{ printf "v-%s-%s-%s-%s" (.RoleName | truncate 64) (.DisplayName | truncate 64) (unix_time) (random 24) | truncate 256 | lowercase }}`
	testRoleUserEmailUpdate      = "me@example.org"
)

func Test_Roles(t *testing.T) {
	t.Run("Roles_SimpleCRUD", testRoles_SimpleCRUD)
	t.Run("Roles_Create_Fail", testRoles_Create_MissingRequireFields)
	t.Run("Roles_Update_Fail", testRoles_Update_Fail)
}

func initBaseAdminConfig(b logical.Backend, s logical.Storage) (*logical.Response, error) {
	return doAction(actionCreate, configAdminPath, b, s, testData{
		"username": testConfigAdminUsername,
		"password": testConfigAdminPassword,
		"url":      testConfigAdminURL,
	})
}

func testRoles_SimpleCRUD(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	// Create base admin config
	resp, err := initBaseAdminConfig(b, reqStorage)
	require.NoError(t, err)
	assert.Nil(t, resp)

	// Create role
	roleData := testData{
		"name":             testRoleName,
		"nexus_roles":      testRoleNexusRoles,
		"user_id_template": testRoleUserIdTemplate,
		"user_email":       testRoleUserEmail,
	}
	resp, err = doAction(actionCreate, rolesPath+testRoleName, b, reqStorage, roleData)
	require.NoError(t, err)
	assert.Nil(t, resp)

	// List roles
	resp, err = doAction(actionList, rolesPath, b, reqStorage, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Data, 1)

	// Read role config to verify the written values
	resp, err = doAction(actionRead, rolesPath+testRoleName, b, reqStorage, nil)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, resp.Error())
	assert.Equal(t, testRoleName, resp.Data["name"])
	assert.Equal(t, []string{"nx-test1", "nx-test2"}, resp.Data["nexus_roles"])
	assert.Equal(t, testRoleUserIdTemplate, resp.Data["user_id_template"]) // default value
	assert.Equal(t, testRoleUserEmail, resp.Data["user_email"])            // default value

	// Update role
	updateData := testData{
		"name":             testRoleName,
		"nexus_roles":      testRoleNexusRolesUpdate,
		"user_id_template": testRoleUserIdTemplateUpdate,
		"user_email":       testRoleUserEmailUpdate,
	}
	resp, err = doAction(actionUpdate, rolesPath+testRoleName, b, reqStorage, updateData)

	require.NoError(t, err)
	assert.Nil(t, resp)

	// Read role again to verify the written values
	resp, err = doAction(actionRead, rolesPath+testRoleName, b, reqStorage, nil)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, resp.Error())
	assert.Equal(t, testRoleName, resp.Data["name"])
	assert.Equal(t, []string{"nx-test1", "nx-test2", "nx-test3"}, resp.Data["nexus_roles"])
	assert.Equal(t, testRoleUserIdTemplateUpdate, resp.Data["user_id_template"])
	assert.Equal(t, testRoleUserEmailUpdate, resp.Data["user_email"])

	// Delete role
	resp, err = doAction(actionDelete, rolesPath+testRoleName, b, reqStorage, nil)
}

func testRoles_Create_MissingRequireFields(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	// Create base admin config
	resp, err := initBaseAdminConfig(b, reqStorage)
	require.NoError(t, err)
	assert.Nil(t, resp)

	testCases := []struct {
		data          *testData // test input data
		expectedError string    // expected missing field in the error message
	}{
		{
			data:          &testData{},
			expectedError: `missing "nexus_roles" in role definition`,
		},
		{
			data: &testData{
				"name": testRoleName,
			},
			expectedError: `missing "nexus_roles" in role definition`,
		},
	}

	for i, tc := range testCases {
		resp, err := doAction(actionCreate, fmt.Sprintf("%s%s-%d", rolesPath, testRoleName, i), b, reqStorage, *tc.data)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Equal(t, tc.expectedError, resp.Error().Error())
	}
}

func testRoles_Update_Fail(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	// Create base admin config
	resp, err := initBaseAdminConfig(b, reqStorage)
	require.NoError(t, err)
	assert.Nil(t, resp)

	// Create role base config
	roleData := testData{
		"name":        testRoleName,
		"nexus_roles": testRoleNexusRoles,
	}
	resp, err = doAction(actionCreate, rolesPath+testRoleName, b, reqStorage, roleData)
	require.NoError(t, err)
	assert.Nil(t, resp)

	testCases := []struct {
		data          *testData // test input data
		expectedError string    // expected missing field in the error message
	}{
		// Invalid user_email
		{
			data: &testData{
				"user_email": "abc",
			},
			expectedError: `"user_email" is not valid`,
		},
		{
			data: &testData{
				"user_email": "abc@",
			},
			expectedError: `"user_email" is not valid`,
		},
		{
			data: &testData{
				"user_email": "abc@xyz",
			},
			expectedError: `"user_email" is not valid`,
		},
		{
			data: &testData{
				"user_email": "abc@xyz.",
			},
			expectedError: `"user_email" is not valid`,
		},
		{
			data: &testData{
				"user_email": "@abc.xyz",
			},
			expectedError: `"user_email" is not valid`,
		},
		// Invalid user_id_template
		{
			data: &testData{
				"user_id_template": "{{ abc",
			},
			expectedError: `unable to initialize "user_id_template"`,
		},
		{
			data: &testData{
				"user_id_template": "bad_{{ .somethingInvalid }}_testing {{",
			},
			expectedError: `unable to initialize "user_id_template"`,
		},
		// ttl greater max_ttl
		{
			data: &testData{
				"ttl":     "30s",
				"max_ttl": "10s",
			},
			expectedError: `"ttl" cannot be greater than "max_ttl"`,
		},
	}

	for _, tc := range testCases {
		resp, _ := doAction(actionUpdate, rolesPath+testRoleName, b, reqStorage, *tc.data)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Equal(t, tc.expectedError, resp.Error().Error())
	}
}
