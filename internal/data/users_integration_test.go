//go:build integration
// +build integration

package data

import (
	"testing"
	"time"

	"github.com/djudju12/greenlight/internal/util"
	"github.com/stretchr/testify/require"
)

func TestUpdateUser(t *testing.T) {
	beforeUser := randomUser()

	err := testModels.Users.Insert(&beforeUser)
	t.Logf("before user: %+v", beforeUser)

	require.NoError(t, err)

	beforeUser.Email = util.RandomEmail()
	beforeUser.Name = util.RandomFullName()

	err = beforeUser.Password.Set(util.RandomPassword())
	require.NoError(t, err)

	versionBeforeUpdate := beforeUser.Version

	t.Logf("before user with new fields: %+v", beforeUser)

	err = testModels.Users.Update(&beforeUser)

	require.NoError(t, err)

	afterUser, err := testModels.Users.GetByEmail(beforeUser.Email)

	t.Logf("after user: %+v", afterUser)

	require.NoError(t, err)
	verifyUsers(t, beforeUser, *afterUser)

	require.NotEqual(t, versionBeforeUpdate, afterUser.Version)
}

func TestGetUserByEmail(t *testing.T) {
	expectedUser := randomUser()
	err := testModels.Users.Insert(&expectedUser)
	t.Logf("expected user: %+v", expectedUser)
	require.NoError(t, err)

	actualUser, err := testModels.Users.GetByEmail(expectedUser.Email)

	t.Logf("actual user: %+v", actualUser)
	require.NoError(t, err)
	verifyUsers(t, expectedUser, *actualUser)
}

func TestInserDuplicateUsesr(t *testing.T) {
	user1 := randomUser()

	err := testModels.Users.Insert(&user1)
	t.Logf("random user1: %+v", user1)

	require.NoError(t, err)

	user2 := randomUser()
	user2.Email = user1.Email

	t.Logf("random user2: %+v", user2)

	err = testModels.Users.Insert(&user2)

	require.Error(t, ErrDuplicateEmail, err)
}

func TestUpdatesDuplicateUsesr(t *testing.T) {
	user1 := randomUser()
	user2 := randomUser()

	err := testModels.Users.Insert(&user1)
	t.Logf("random user1: %+v", user1)

	require.NoError(t, err)

	err = testModels.Users.Insert(&user2)
	t.Logf("random user2: %+v", user2)
	require.NoError(t, err)

	user2.Email = user1.Email

	err = testModels.Users.Update(&user2)
	require.Error(t, ErrDuplicateEmail, err)
}

func TestInsertUser(t *testing.T) {
	user := randomUser()
	err := testModels.Users.Insert(&user)
	t.Logf("random user:  %+v", user)

	require.NoError(t, err)
}

func TestTokensForUsers(t *testing.T) {
	user := randomUser()

	testModels.Users.Insert(&user)
	t.Logf("random user: %+v", user)

	token1, err := testModels.Tokens.New(user.ID, time.Hour, ScopeActiviation)

	t.Logf("token1: %+v", token1)
	require.NoError(t, err)

	token2, err := testModels.Tokens.New(user.ID, time.Hour, ScopeAuthentication)

	t.Logf("token2: %+v", token2)
	require.NoError(t, err)

	actualUser1, err := testModels.Users.GetForToken(ScopeActiviation, token1.Plaintext)

	t.Logf("user returned for token1: %+v", actualUser1)
	verifyUsers(t, user, *actualUser1)
	require.NoError(t, err)

	actualUser2, err := testModels.Users.GetForToken(ScopeAuthentication, token2.Plaintext)

	t.Logf("user returned for token2: %+v", actualUser2)
	require.NoError(t, err)
	verifyUsers(t, user, *actualUser2)

	err = testModels.Tokens.DeleteAllForUser(ScopeActiviation, user.ID)
	require.NoError(t, err)

	err = testModels.Tokens.DeleteAllForUser(ScopeAuthentication, user.ID)
	require.NoError(t, err)

	_, err = testModels.Users.GetForToken(ScopeActiviation, token1.Plaintext)
	require.ErrorIs(t, ErrRecordNotFound, err)

	_, err = testModels.Users.GetForToken(ScopeActiviation, token2.Plaintext)
	require.ErrorIs(t, ErrRecordNotFound, err)
}

func TestPermissionsForUser(t *testing.T) {
	user := randomUser()
	err := testModels.Users.Insert(&user)
	t.Logf("random user: %+v", user)

	require.NoError(t, err)

	expectedPermissions := Permissions{
		"movies:read",
		"movies:write",
	}

	for _, permission := range expectedPermissions {
		t.Logf("inserting %s permission", permission)
		err = testModels.Permissions.AddForUser(user.ID, permission)

		require.NoError(t, err)
	}

	actualPermissions, err := testModels.Permissions.GetAllForUser(user.ID)
	t.Logf("permission: expected %+v | actual %+v", expectedPermissions, actualPermissions)

	require.NoError(t, err)
	require.ElementsMatch(t, expectedPermissions, actualPermissions)
}

func verifyUsers(t *testing.T, expectedUser User, actualUser User) {
	require.NotEmpty(t, actualUser)

	require.NotZero(t, actualUser.ID)
	require.NotZero(t, actualUser.Version)

	require.Equal(t, expectedUser.Name, actualUser.Name)
	require.Equal(t, expectedUser.Email, actualUser.Email)
	require.Equal(t, expectedUser.Password.hash, actualUser.Password.hash)
	require.Equal(t, expectedUser.Activated, actualUser.Activated)

	require.WithinDuration(t, time.Now(), actualUser.CreatedAt, time.Second)
}

func randomUser() User {
	var pwd password
	pwd.Set(util.RandomPassword())
	return User{
		Name:      util.RandomFullName(),
		Email:     util.RandomEmail(),
		Password:  pwd,
		Activated: true,
	}
}
