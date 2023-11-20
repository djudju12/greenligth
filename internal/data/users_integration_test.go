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
	t.Log("running 'TestUpdateUser'")

	randomUser1 := randomUser()
	err := testModels.Users.Insert(&randomUser1)

	require.NoError(t, err)

	beforeUser, err := testModels.Users.GetByEmail(randomUser1.Email)

	t.Logf("user created before update: %+v", beforeUser)

	require.NoError(t, err)
	require.NotEmpty(t, beforeUser)

	beforeUser.Email = util.RandomEmail()
	beforeUser.Name = util.RandomFullName()
	versionBeforeUpdate := beforeUser.Version
	err = beforeUser.Password.Set(util.RandomPassword())

	t.Logf("before update user with new fields: %+v", beforeUser)

	require.NoError(t, err)

	t.Log("making updates...")
	err = testModels.Users.Update(beforeUser)

	require.NoError(t, err)

	t.Logf("getting updated user by email. Email: %s", beforeUser.Email)
	afterUser, err := testModels.Users.GetByEmail(beforeUser.Email)

	t.Logf("actual user: %+v", afterUser)
	require.NoError(t, err)
	require.NotEmpty(t, afterUser)

	require.Equal(t, beforeUser.Name, afterUser.Name)
	require.Equal(t, beforeUser.Email, afterUser.Email)
	require.Equal(t, beforeUser.Password.hash, afterUser.Password.hash)
	require.Equal(t, beforeUser.Activated, afterUser.Activated)
	require.Equal(t, beforeUser.ID, afterUser.ID)

	require.Equal(t, beforeUser.CreatedAt, afterUser.CreatedAt)
	require.NotEqual(t, versionBeforeUpdate, afterUser.Version)
}

func TestGetUserByEmail(t *testing.T) {
	t.Log("running 'TestGetUserByEmail'")
	expectedUser := randomUser()
	err := testModels.Users.Insert(&expectedUser)
	require.NoError(t, err)

	t.Logf("geting user by email. Email: %s", expectedUser.Email)
	user, err := testModels.Users.GetByEmail(expectedUser.Email)

	t.Logf("actual user: %+v", user)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.Version)

	require.Equal(t, expectedUser.Name, user.Name)
	require.Equal(t, expectedUser.Email, user.Email)
	require.Equal(t, expectedUser.Password.hash, user.Password.hash)
	require.Equal(t, expectedUser.Activated, user.Activated)

	require.WithinDuration(t, time.Now(), user.CreatedAt, time.Second)
}

func TestGetForToken(t *testing.T) {
	expectedUser := randomUser()

	err := testModels.Users.Insert(&expectedUser)
	require.NoError(t, err)

	tokenDuration := time.Minute
	token, err := testModels.Tokens.New(expectedUser.ID, tokenDuration, ScopeActiviation)
	require.NoError(t, err)

	actualUser, err := testModels.Users.GetForToken(ScopeActiviation, token.Plaintext)

	require.NoError(t, err)
	require.NotEmpty(t, actualUser)

	require.NotZero(t, actualUser.ID)
	require.NotZero(t, actualUser.Version)

	require.Equal(t, expectedUser.Name, actualUser.Name)
	require.Equal(t, expectedUser.Email, actualUser.Email)
	require.Equal(t, expectedUser.Password.hash, actualUser.Password.hash)
	require.Equal(t, expectedUser.Activated, actualUser.Activated)

	require.WithinDuration(t, time.Now(), actualUser.CreatedAt, time.Second)
}

func TestInserDuplicateUsesr(t *testing.T) {
	user1 := randomUser()
	err := testModels.Users.Insert(&user1)
	require.NoError(t, err)

	user2 := randomUser()
	user2.Email = user1.Email
	err = testModels.Users.Insert(&user2)

	require.Error(t, ErrDuplicateEmail, err)
}

func TestUpdatesDuplicateUsesr(t *testing.T) {
	user1 := randomUser()
	user2 := randomUser()

	err := testModels.Users.Insert(&user1)
	require.NoError(t, err)

	err = testModels.Users.Insert(&user2)
	require.NoError(t, err)

	user2.Email = user1.Email

	err = testModels.Users.Update(&user2)
	require.Error(t, ErrDuplicateEmail, err)
}

func TestInsertUser(t *testing.T) {
	t.Log("running 'TestInsertUser'")
	user := randomUser()

	t.Logf("inserting random user:  %+v", user)
	err := testModels.Users.Insert(&user)

	require.NoError(t, err)
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
