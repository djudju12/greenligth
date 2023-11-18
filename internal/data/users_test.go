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
	// given
	t.Log("running 'TestUpdateUser'")

	randomUser1 := randomUser()
	err := InsertUser(&randomUser1)

	require.NoError(t, err)

	beforeUser, err := testModels.Users.GetByEmail(randomUser1.Email)

	t.Logf("user created before update: %+v", beforeUser)

	require.NoError(t, err)
	require.NotEmpty(t, beforeUser)

	// when
	beforeUser.Email = util.RandomEmail()
	beforeUser.Name = util.RandomFullName()
	versionBeforeUpdate := beforeUser.Version
	err = beforeUser.Password.Set(util.RandomPassword())

	t.Logf("before update user with new fields: %+v", beforeUser)

	require.NoError(t, err)

	t.Log("making updates...s")
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
	// given
	t.Log("running 'TestGetUserByEmail'")
	expectedUser := randomUser()
	err := InsertUser(&expectedUser)
	require.NoError(t, err)

	// when
	t.Logf("geting user by email. Email: %s", expectedUser.Email)
	user, err := testModels.Users.GetByEmail(expectedUser.Email)

	// then
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

func TestInsertUser(t *testing.T) {
	// given
	t.Log("running 'TestInsertUser'")
	user := randomUser()

	// when
	t.Logf("inserting random user:  %+v", user)
	err := InsertUser(&user)

	// then
	require.NoError(t, err)

}

func InsertUser(user *User) error {
	return testModels.Users.Insert(user)
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
