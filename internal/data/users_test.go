//go:build integration
// +build integration

package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInsertUser(t *testing.T) {
	// given
	user := &User{
		Name:  "Jonathan",
		Email: "jonathan.willian321@mail.com",
		Password: password{
			hash: []byte("1234"),
		},
		Activated: true,
	}
	// when
	err := testModels.Users.Insert(user)

	// then
	require.NoError(t, err)
}
