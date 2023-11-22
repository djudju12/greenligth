package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/djudju12/greenlight/internal/data"
	"github.com/djudju12/greenlight/internal/jsonlog"
	mockdb "github.com/djudju12/greenlight/internal/mocks"
	"github.com/djudju12/greenlight/internal/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type test struct {
	app      *application
	recorder *httptest.ResponseRecorder
	url      string

	// so we can pass this to the logger and not write direct to
	// stdout. This is a filed so we can close the file after
	// test
	tempFile *os.File
}

func newTest(t *testing.T, url string) test {
	ctrl := gomock.NewController(t)
	recorder := httptest.NewRecorder()

	users := mockdb.NewMockUserQuerier(ctrl)
	mailer := mockdb.NewMockMailer(ctrl)
	permissions := mockdb.NewMockPermissionQuerier(ctrl)
	tokens := mockdb.NewMockTokenQuerier(ctrl)

	f, err := os.CreateTemp("", "tmpfile-")
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		models: &data.Models{
			Users:       users,
			Permissions: permissions,
			Tokens:      tokens,
		},
		logger: jsonlog.New(f, jsonlog.LevelInfo),
		mailer: mailer,
	}

	return test{
		recorder: recorder,
		url:      url,
		app:      app,
		tempFile: f,
	}
}

func (t *test) close() {
	t.tempFile.Close()
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
			name: "User Register Handle - 201 CREATED",
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
		{
			name: "User Register Handle - 422 PASSWORD WITH MORE THAN 72 CARACTERES",
			requestBody: RegisterUserRequest{
				Name:     expectedUser.Name,
				Email:    expectedUser.Email,
				Password: util.RandomString(73),
			},
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no stubs to build in this test")
			},

			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
		{
			name: "User Register Handle - 422 INVALID USER",
			requestBody: RegisterUserRequest{
				Name:     "",
				Email:    "",
				Password: plaintextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no stubs to build in this test")
			},

			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
		{
			name: "User Register Handle - 422 DUPLICATED EMAIL",
			requestBody: RegisterUserRequest{
				Name:     expectedUser.Name,
				Email:    expectedUser.Email,
				Password: plaintextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, _, _ := modelMocks(t, app.models)

				mockUsers.EXPECT().
					Insert(EqUserInsert(expectedUser, plaintextPassword)).
					Return(data.ErrDuplicateEmail)

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
		{
			name: "User Register Handle - 500 DB RETURNED ERROR IN ADD PERMISSION",
			requestBody: RegisterUserRequest{
				Name:     expectedUser.Name,
				Email:    expectedUser.Email,
				Password: plaintextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, mockPermissions, _ := modelMocks(t, app.models)

				mockUsers.EXPECT().
					Insert(EqUserInsert(expectedUser, plaintextPassword)).
					DoAndReturn(func(user *data.User) error {
						user.ID = expectedUser.ID
						user.CreatedAt = expectedUser.CreatedAt
						user.Version = expectedUser.Version
						return nil
					})

				mockPermissions.EXPECT().
					AddForUser(gomock.Any(), gomock.Any()).
					Return(errors.New("some error"))

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "User Register Handle - 500 DB RETURNED ERROR IN INSERT USERS",
			requestBody: RegisterUserRequest{
				Name:     expectedUser.Name,
				Email:    expectedUser.Email,
				Password: plaintextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, _, _ := modelMocks(t, app.models)

				mockUsers.EXPECT().
					Insert(EqUserInsert(expectedUser, plaintextPassword)).
					Return(errors.New("some error"))

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "User Register Handle - 500 DB RETURNED ERROR IN NEW TOKEN",
			requestBody: RegisterUserRequest{
				Name:     expectedUser.Name,
				Email:    expectedUser.Email,
				Password: plaintextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, mockPermissions, mockTokens := modelMocks(t, app.models)

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
					Return(&data.Token{}, errors.New("some error"))

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "User Register Handle - 201 MAILER FAILING SHOULD NOT HAVE EFFECT ON USER CREATION",
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
					Send(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("some error"))

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, r.Code)
				requireBodyMatchUser(t, r.Body, &expectedUser)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			// url doesnt make any differece here
			// but i think its good to make explicit
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

			test.close()
		})
	}
}

func TestActiveteUserHandle(t *testing.T) {
	expectedUser, _ := randomUser()
	expetedToken := randomToken()
	testCases := []struct {
		name          string
		requestBody   ActiveteUserRequest
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Activate User Handler - 200 OK",
			requestBody: ActiveteUserRequest{
				TokenPlainText: expetedToken.Plaintext,
			},
			buildStubs: func(t *testing.T, app *application) {
				users, _, tokens := modelMocks(t, app.models)

				users.EXPECT().
					GetForToken(data.ScopeActiviation, expetedToken.Plaintext).
					Return(&expectedUser, nil)

				// i dont know if this is any good :p
				users.EXPECT().
					Update(gomock.Any()).
					DoAndReturn(func(user *data.User) error {
						require.True(t, user.Activated)
						return nil
					})

				tokens.EXPECT().
					DeleteAllForUser(data.ScopeActiviation, expectedUser.ID).
					Return(nil)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, r.Code)
				requireBodyMatchUser(t, r.Body, &expectedUser)
			},
		},
		{
			name: "Activate User Handler - 422 TOKEN TOO SMOL PLAIN TEXT",
			requestBody: ActiveteUserRequest{
				TokenPlainText: util.RandomString(10),
			},
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no stubs to build in this test")
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
		{
			name: "Activate User Handler - 422 TOKEN INVALID OR EXPIRED",
			requestBody: ActiveteUserRequest{
				TokenPlainText: expetedToken.Plaintext,
			},
			buildStubs: func(t *testing.T, app *application) {
				users, _, _ := modelMocks(t, app.models)

				users.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(&data.User{}, data.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
		{
			name: "Activate User Handler - 500 DB RETURNED ERROR ON GET FOR TOKEN",
			requestBody: ActiveteUserRequest{
				TokenPlainText: expetedToken.Plaintext,
			},
			buildStubs: func(t *testing.T, app *application) {
				users, _, _ := modelMocks(t, app.models)

				users.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(&data.User{}, errors.New("DB RETURNED ERROR ON GET FOR TOKEN"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "Activate User Handler - 409 DB EDIT CONFLICT ON USER UPDATE",
			requestBody: ActiveteUserRequest{
				TokenPlainText: expetedToken.Plaintext,
			},
			buildStubs: func(t *testing.T, app *application) {
				users, _, _ := modelMocks(t, app.models)

				users.EXPECT().
					GetForToken(data.ScopeActiviation, expetedToken.Plaintext).
					Return(&expectedUser, nil)

				// i dont know if this is any good :p
				users.EXPECT().
					Update(gomock.Any()).
					Return(data.ErrEditConflict)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, r.Code)
			},
		},
		{
			name: "Activate User Handler - 500 DB RETURNED ERROR ON USER UPDATE",
			requestBody: ActiveteUserRequest{
				TokenPlainText: expetedToken.Plaintext,
			},
			buildStubs: func(t *testing.T, app *application) {
				users, _, _ := modelMocks(t, app.models)

				users.EXPECT().
					GetForToken(data.ScopeActiviation, expetedToken.Plaintext).
					Return(&expectedUser, nil)

				// i dont know if this is any good :p
				users.EXPECT().
					Update(gomock.Any()).
					Return(errors.New("RETURNED ERROR ON USER UPDATE"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "Activate User Handler - 500 DB RETURNED ERROR ON DELETE FOR ALL USERS",
			requestBody: ActiveteUserRequest{
				TokenPlainText: expetedToken.Plaintext,
			},
			buildStubs: func(t *testing.T, app *application) {
				users, _, tokens := modelMocks(t, app.models)

				users.EXPECT().
					GetForToken(gomock.Any(), gomock.Any()).
					Return(&expectedUser, nil)

				// i dont know if this is any good :p
				users.EXPECT().
					Update(gomock.Any()).
					Return(nil)

				tokens.EXPECT().
					DeleteAllForUser(gomock.Any(), gomock.Any()).
					Return(errors.New("RETURNED ERROR ON DELETE FOR ALL USERS"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
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

			request := httptest.NewRequest(http.MethodPut, test.url, body)

			// when
			test.app.activateUserHandle(test.recorder, request)

			// then
			tc.checkResponse(t, test.recorder)

			test.close()
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
		Plaintext: util.RandomString(26),
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

type eqUserInsertMatcher struct {
	user              data.User
	plainTextPassword string
}

func (eq eqUserInsertMatcher) Matches(x any) bool {
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

func (eq eqUserInsertMatcher) String() string {
	return fmt.Sprintf("matchs arg %+v and password %+v", eq.user, eq.plainTextPassword)
}

func EqUserInsert(user data.User, plainTextPassword string) gomock.Matcher {
	return eqUserInsertMatcher{user, plainTextPassword}
}
