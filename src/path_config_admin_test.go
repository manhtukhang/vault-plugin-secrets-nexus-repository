package nxr

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testConfigAdminUsername       = "admin"
	testConfigAdminPassword       = "Testing!123"
	testConfigAdminURL            = "http://localhost:1234"
	testConfigAdminInsecure       = false
	testConfigAdminTimeout        = 30
	testConfigAdminUsernameUpdate = "admin-new"
	testConfigAdminPasswordUpdate = "Testing!123-new"
	testConfigAdminURLUpdate      = "http://localhost:1235"
	testConfigAdminInsecureUpdate = true
	testConfigAdminTimeoutUpdate  = 60
)

func Test_ConfigAdmin(t *testing.T) {
	t.Run("ConfigAdmin_SimpleCRUD", testConfigAdmin_SimpleCRUD)
	t.Run("ConfigAdmin_Create", testConfigAdmin_Create)
	t.Run("ConfigAdmin_Create_Fail", testConfigAdmin_Create_MissingRequireFields)
	t.Run("ConfigAdmin_Update", testConfigAdmin_Update)
	t.Run("ConfigAdmin_Update_Fail", testConfigAdmin_Update_Fail)
	t.Run("ConfigAdmin_ReadDelete_Empty", testConfigAdmin_ReadDelete_Empty)
}

func testConfigAdmin_SimpleCRUD(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	// Create base admin config
	initData := testData{
		"username": testConfigAdminUsername,
		"password": testConfigAdminPassword,
		"url":      testConfigAdminURL,
		"insecure": testConfigAdminInsecure,
		"timeout":  testConfigAdminTimeout,
	}
	resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, initData)

	require.NoError(t, err)
	assert.Nil(t, resp)

	// Read admin config to verify the written values
	resp, err = doAction(actionRead, configAdminPath, b, reqStorage, nil)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, resp.Error())
	assert.Equal(t, testConfigAdminUsername, resp.Data["username"])
	assert.Equal(t, testConfigAdminURL, resp.Data["url"])
	assert.Equal(t, testConfigAdminInsecure, resp.Data["insecure"])
	assert.Equal(t, testConfigAdminTimeout, resp.Data["timeout"])

	// Update admin config
	updateData := testData{
		"username": testConfigAdminUsernameUpdate,
		"password": testConfigAdminPasswordUpdate,
		"url":      testConfigAdminURLUpdate,
		"insecure": testConfigAdminInsecureUpdate,
		"timeout":  testConfigAdminTimeoutUpdate,
	}
	resp, err = doAction(actionCreate, configAdminPath, b, reqStorage, updateData)

	require.NoError(t, err)
	assert.Nil(t, resp)

	// Read admin config again to verify the updated values
	resp, err = doAction(actionRead, configAdminPath, b, reqStorage, nil)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, resp.Error())
	assert.Equal(t, testConfigAdminUsernameUpdate, resp.Data["username"])
	assert.Equal(t, testConfigAdminURLUpdate, resp.Data["url"])
	assert.Equal(t, testConfigAdminInsecureUpdate, resp.Data["insecure"])
	assert.Equal(t, testConfigAdminTimeoutUpdate, resp.Data["timeout"])

	// Delete admin config
	resp, err = doAction(actionDelete, configAdminPath, b, reqStorage, nil)
	require.NoError(t, err)
	assert.Nil(t, resp)
}

func testConfigAdmin_Create(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	// Create admin config with minimum required fields
	data := testData{
		"username": testConfigAdminUsername,
		"password": testConfigAdminPassword,
		"url":      testConfigAdminURL,
	}
	resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, data)

	require.NoError(t, err)
	assert.Nil(t, resp)

	// Read admin config to verify the written values
	resp, err = doAction(actionRead, configAdminPath, b, reqStorage, nil)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	require.NoError(t, resp.Error())
	assert.Equal(t, testConfigAdminUsername, resp.Data["username"])
	assert.Equal(t, testConfigAdminURL, resp.Data["url"])
	assert.Equal(t, testConfigAdminInsecure, resp.Data["insecure"]) // default value
	assert.Equal(t, testConfigAdminTimeout, resp.Data["timeout"])   // default value
}

func testConfigAdmin_Create_MissingRequireFields(t *testing.T) {
	testCases := []struct {
		data         *testData // test input data
		missingField string    // expected missing field in the error message
	}{
		{
			data:         &testData{},
			missingField: "username",
		},
		{
			data: &testData{
				"username": testConfigAdminUsername,
			},
			missingField: "url",
		},
		{
			data: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
			},
			missingField: "password",
		},
		{
			data: &testData{
				"username": testConfigAdminUsername,
				"password": testConfigAdminPassword,
			},
			missingField: "url",
		},
	}

	for _, tc := range testCases {
		b, reqStorage := getTestBackend(t)
		resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, *tc.data)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Equal(t, fmt.Sprintf(`missing "%s" in admin configuration`, tc.missingField), resp.Error().Error())
	}
}

func testConfigAdmin_Update(t *testing.T) {
	//
	testCases := []struct {
		updateData *testData // test input data
		expected   *testData // expected response data
	}{
		// Update "username"
		{
			updateData: &testData{
				"username": testConfigAdminUsernameUpdate,
			},
			expected: &testData{
				"username": testConfigAdminUsernameUpdate, // this field will be changed
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeout,
			},
		},
		// Update "password"
		{
			updateData: &testData{
				"password": testConfigAdminPasswordUpdate,
			},
			// Nothing will be changed
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeout,
			},
		},
		// Update "url" (and "password" must be set also)
		{
			updateData: &testData{
				"url":      testConfigAdminURLUpdate,
				"password": testConfigAdminPasswordUpdate,
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURLUpdate, // this field will be changed
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeout,
			},
		},
		// Update "insecure"
		{
			updateData: &testData{
				"insecure": testConfigAdminInsecureUpdate,
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecureUpdate, // this field will be changed
				"timeout":  testConfigAdminTimeout,
			},
		},
		{
			updateData: &testData{
				"insecure": "true", // string of bool value, returns true
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecureUpdate, // this field will be changed
				"timeout":  testConfigAdminTimeout,
			},
		},
		{
			updateData: &testData{
				"insecure": "True", // string of bool value, returns true
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecureUpdate, // this field will be changed
				"timeout":  testConfigAdminTimeout,
			},
		},
		{
			updateData: &testData{
				"insecure": "TRUE", // string of bool value, returns true
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecureUpdate, // this field will be changed
				"timeout":  testConfigAdminTimeout,
			},
		},
		{
			updateData: &testData{
				"insecure": 0, // number as bool value, returns false
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": false, // this field will be changed
				"timeout":  testConfigAdminTimeout,
			},
		},
		{
			updateData: &testData{
				"insecure": 1, // number as bool value, returns true
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecureUpdate, // this field will be changed
				"timeout":  testConfigAdminTimeout,
			},
		},
		{
			updateData: &testData{
				"insecure": -1, // number as bool value, returns true
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecureUpdate, // this field will be changed
				"timeout":  testConfigAdminTimeout,
			},
		},
		// Update "timeout"
		{
			updateData: &testData{
				"timeout": testConfigAdminTimeoutUpdate,
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeoutUpdate, // this field will be changed
			},
		},
		{
			updateData: &testData{
				"timeout": 60.1, // floating value, returns 60 (remove decima point)
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeoutUpdate, // this field will be changed
			},
		},
		{
			updateData: &testData{
				"timeout": 60.9, // floating value, rerurns 60 (remove decima point)
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeoutUpdate, // this field will be changed
			},
		},
		{
			updateData: &testData{
				"timeout": "60", // string of int value, returns 60
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeoutUpdate, // this field will be changed
			},
		},
		{
			updateData: &testData{
				"timeout": "60s", // string of time duration, returns 60
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeoutUpdate, // this field will be changed
			},
		},
		{
			updateData: &testData{
				"timeout": "1m", // string of time duration (1 minute), returns 60
			},
			expected: &testData{
				"username": testConfigAdminUsername,
				"url":      testConfigAdminURL,
				"insecure": testConfigAdminInsecure,
				"timeout":  testConfigAdminTimeoutUpdate, // this field will be changed
			},
		},
	}

	for _, tc := range testCases {
		// Create base config
		b, reqStorage := getTestBackend(t)

		initData := testData{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      testConfigAdminURL,
		}
		resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, initData)

		require.NoError(t, err)
		assert.Nil(t, resp)

		// Update
		resp, err = doAction(actionUpdate, configAdminPath, b, reqStorage, *tc.updateData)

		require.NoError(t, err)
		assert.Nil(t, resp)
		assert.False(t, resp.IsError())

		// Read for verifying
		resp, err = doAction(actionRead, configAdminPath, b, reqStorage, nil)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.IsError())
		assert.Equal(t, len(*tc.expected), len(resp.Data))

		for k, expectedV := range *tc.expected {
			actualV, ok := resp.Data[k]
			assert.True(t, ok)
			assert.Equal(t, expectedV, actualV)
		}
	}
}

func testConfigAdmin_Update_Fail(t *testing.T) {
	testCases := []struct {
		updateData            *testData // test input data
		expectedErrorContains string    // expected response data
	}{
		// Update only "url" (keep "password" unchange)
		{
			updateData: &testData{
				"url": testConfigAdminURLUpdate,
			},
			expectedErrorContains: `missing "password" in admin configuration`,
		},
		// Update "insecure" with wrong formats
		{
			updateData: &testData{
				"insecure": "no",
			},
			expectedErrorContains: `Field validation failed: error converting input .* for field "insecure": cannot parse '' as bool: strconv.ParseBool: parsing .*: invalid syntax`,
		},
		{
			updateData: &testData{
				"insecure": "TrUe",
			},
			expectedErrorContains: `Field validation failed: error converting input .* for field "insecure": cannot parse '' as bool: strconv.ParseBool: parsing .*: invalid syntax`,
		},
		{
			updateData: &testData{
				"timeout": "60.1",
			},
			expectedErrorContains: `Field validation failed: error converting input .* for field "timeout": time: missing unit in duration .*`,
		},
	}

	for _, tc := range testCases {
		// Create base config
		b, reqStorage := getTestBackend(t)

		initData := testData{
			"username": testConfigAdminUsername,
			"password": testConfigAdminPassword,
			"url":      testConfigAdminURL,
		}
		resp, err := doAction(actionCreate, configAdminPath, b, reqStorage, initData)

		require.NoError(t, err)
		assert.Nil(t, resp)
		assert.False(t, resp.IsError())

		// Update
		resp, err = doAction(actionUpdate, configAdminPath, b, reqStorage, *tc.updateData)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Regexp(t, regexp.MustCompile(tc.expectedErrorContains), resp.Error().Error())
	}
}

func testConfigAdmin_ReadDelete_Empty(t *testing.T) {
	b, reqStorage := getTestBackend(t)

	expectedError := `admin configuration not found`

	for _, action := range []logical.Operation{actionRead, actionDelete} {
		resp, err := doAction(action, configAdminPath, b, reqStorage, nil)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Equal(t, expectedError, resp.Error().Error())
	}
}
