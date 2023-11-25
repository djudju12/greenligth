package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/djudju12/greenlight/internal/data"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateAutheticationTokenHandler(t *testing.T) {
	user, plainTextPassword := randomUser()
	err := user.Password.Set(plainTextPassword)
	require.NoError(t, err)

	token := randomToken()

	testCases := []struct {
		name          string
		requestBody   CreateAuthenticationRequest
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Test Create Authentication Token - 201 CREATED",
			requestBody: CreateAuthenticationRequest{
				Email:    user.Email,
				Password: plainTextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, _, mockTokens := modelMocks(t, app.models)

				mockUsers.EXPECT().
					GetByEmail(user.Email).
					Return(&user, nil)

				mockTokens.EXPECT().
					New(user.ID, 24*time.Hour, data.ScopeAuthentication).
					Return(token, nil)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, r.Code)
				requireMatchTokens(t, r.Body, token)
			},
		},
		{
			name:        "Test Create Authentication Token - 422 INVALID PARAMETERS",
			requestBody: CreateAuthenticationRequest{},
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no need for stubs in the test")
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
		{
			name: "Test Create Authentication Token - 500 DB RETURN ERROR ON GET EMAIL",
			requestBody: CreateAuthenticationRequest{
				Email:    user.Email,
				Password: plainTextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, _, _ := modelMocks(t, app.models)

				mockUsers.EXPECT().
					GetByEmail(user.Email).
					Return(&data.User{}, errors.New("DB RETURN ERROR ON GET EMAIL"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "Test Create Authentication Token - 500 DB RETURN ERROR ON TOKEN NEW",
			requestBody: CreateAuthenticationRequest{
				Email:    user.Email,
				Password: plainTextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, _, mockTokens := modelMocks(t, app.models)

				mockUsers.EXPECT().
					GetByEmail(user.Email).
					Return(&user, nil)

				mockTokens.EXPECT().
					New(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&data.Token{}, errors.New("DB RETURN ERROR ON TOKEN NEW"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "Test Create Authentication Token - 422 DB RETURN RECORD NOT FOUND",
			requestBody: CreateAuthenticationRequest{
				Email:    user.Email,
				Password: plainTextPassword,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, _, _ := modelMocks(t, app.models)

				mockUsers.EXPECT().
					GetByEmail(user.Email).
					Return(&data.User{}, data.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, r.Code)
			},
		},
		{
			name: "Test Create Authentication Token - 401 PASSWORDS DONT MATCH",
			requestBody: CreateAuthenticationRequest{
				Email:    user.Email,
				Password: plainTextPassword + "invalid",
			},
			buildStubs: func(t *testing.T, app *application) {
				mockUsers, _, _ := modelMocks(t, app.models)

				mockUsers.EXPECT().
					GetByEmail(user.Email).
					Return(&user, nil)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, r.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newUsersTest(t, "/v1/tokens/auth")
			tc.buildStubs(t, test.app)

			body, err := toReader(tc.requestBody)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, test.url, body)

			// when
			test.app.createAuthenticationTokenHandler(test.recorder, request)

			// then
			tc.checkResponse(t, test.recorder)
		})
	}
}

func requireMatchTokens(t *testing.T, body *bytes.Buffer, token *data.Token) {
	bytea, err := io.ReadAll(body)
	require.NoError(t, err)

	var envelope map[string]*data.Token
	err = json.Unmarshal(bytea, &envelope)
	require.NoError(t, err)

	t.Logf("envolpe %+v", envelope)

	gotToken, ok := envelope["authentication_token"]
	require.True(t, ok)

	t.Logf("gotToken %+v | token %+v", gotToken, token)
	require.Equal(t, token.Plaintext, gotToken.Plaintext)
	require.WithinDuration(t, token.Expiry, gotToken.Expiry, time.Second)
}
