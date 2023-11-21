package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/djudju12/greenlight/internal/data"
	mockdb "github.com/djudju12/greenlight/internal/mocks"
	"github.com/djudju12/greenlight/internal/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type test struct {
	app      *application
	recorder *httptest.ResponseRecorder
	url      string
}

func newTest(t *testing.T, url string) test {
	ctrl := gomock.NewController(t)
	recorder := httptest.NewRecorder()
	users := mockdb.NewMockUserQuerier(ctrl)
	mailer := mockdb.NewMockMailer(ctrl)
	permissions := mockdb.NewMockPermissionQuerier(ctrl)
	tokens := mockdb.NewMockTokenQuerier(ctrl)

	app := &application{
		models: &data.Models{
			Users:       users,
			Permissions: permissions,
			Tokens:      tokens,
		},
		mailer: mailer,
	}

	return test{
		recorder: recorder,
		url:      url,
		app:      app,
	}
}

func TestRegisterUserHandle(t *testing.T) {
	expectedUser, plaintextPassword := randomUser()
	expectedToken := randomToken()

	testCases := []struct {
		name          string
		requestBody   RegisterUserRequest
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "User Register Handle - 200 OK",
			requestBody: RegisterUserRequest{
				Name:     expectedUser.Name,
				Email:    expectedUser.Email,
				Password: plaintextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, mockPermissions, mockTokens := modelMocks(t, app.models)
				mockMailer, ok := app.mailer.(*mockdb.MockMailer)
				require.True(t, ok)

				mockUsers.EXPECT().
					Insert(EqUserInsert(expectedUser, plaintextPassword)).
					DoAndReturn(func(user *data.User) error {
						user.ID = expectedUser.ID
						user.CreatedAt = expectedUser.CreatedAt
						user.Version = expectedUser.Version
						return nil
					})

				mockPermissions.EXPECT().
					AddForUser(gomock.Any(), "movies:read").
					Return(nil)

				mockTokens.EXPECT().
					New(expectedUser.ID, 3*24*time.Hour, data.ScopeActiviation).
					Return(expectedToken, nil)

				mockMailer.EXPECT().
					Send(expectedUser.Email, "user_welcome.go.tmpl", map[string]any{
						"activationToken": expectedToken.Plaintext,
						"userID":          expectedUser.ID,
					}).
					Return(nil)

			},

			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, r.Code)
				requireBodyMatchUser(t, r.Body, &expectedUser)
			},
		},
		{
			name:        "User Register Handle - 422 REQUEST WITH EMPTY BODY",
			requestBody: RegisterUserRequest{},
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no stubs to build in this test")
			},

			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newTest(t, "/v1/users")
			tc.buildStubs(t, test.app)

			body, err := toReader(tc.requestBody)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, test.url, body)

			// when
			test.app.registerUserHandle(test.recorder, request)

			// then
			tc.checkResponse(t, test.recorder)
			test.app.wg.Wait()
		})
	}
}

func randomUser() (data.User, string) {
	pwd := util.RandomPassword()
	user := data.User{
		ID:        util.RandomInt(1, 2000),
		Name:      util.RandomFullName(),
		Email:     util.RandomEmail(),
		CreatedAt: time.Now(),
		Activated: false,
	}

	return user, pwd
}

func randomToken() *data.Token {
	return &data.Token{
		Plaintext: util.RandomString(20),
	}
}

func toReader(body any) (io.Reader, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

func modelMocks(t *testing.T, models *data.Models) (
	*mockdb.MockUserQuerier,
	*mockdb.MockPermissionQuerier,
	*mockdb.MockTokenQuerier,
) {
	users, ok := models.Users.(*mockdb.MockUserQuerier)
	require.True(t, ok)

	permissions, ok := models.Permissions.(*mockdb.MockPermissionQuerier)
	require.True(t, ok)

	tokens, ok := models.Tokens.(*mockdb.MockTokenQuerier)
	require.True(t, ok)

	return users, permissions, tokens
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user *data.User) {
	bytea, err := io.ReadAll(body)
	require.NoError(t, err)

	var envelope map[string]*data.User
	err = json.Unmarshal(bytea, &envelope)
	require.NoError(t, err)

	t.Logf("envelope %+v", envelope)

	gotUser, ok := envelope["user"]
	require.True(t, ok)

	t.Logf("gotUser %+v | user %+v", gotUser, user)

	require.Equal(t, user.ID, gotUser.ID)
	require.Equal(t, user.Email, gotUser.Email)
	require.Equal(t, user.Name, gotUser.Name)
	require.Equal(t, user.Activated, gotUser.Activated)
	require.WithinDuration(t, user.CreatedAt, gotUser.CreatedAt, time.Second)
}

type eqUserMatcher struct {
	user              data.User
	plainTextPassword string
}

func (eq eqUserMatcher) Matches(x any) bool {
	user, ok := x.(*data.User)
	if !ok {
		return false
	}

	ok, err := user.Password.Matches(eq.plainTextPassword)
	if err != nil || !ok {
		return false
	}

	user.Password = eq.user.Password

	return user.Activated == eq.user.Activated &&
		user.Email == eq.user.Email &&
		user.Name == eq.user.Name
}

func (eq eqUserMatcher) String() string {
	return fmt.Sprintf("matchs arg %+v and password %+v", eq.user, eq.plainTextPassword)
}

func EqUserInsert(user data.User, plainTextPassword string) gomock.Matcher {
	return eqUserMatcher{user, plainTextPassword}
}
