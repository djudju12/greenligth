package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/djudju12/greenlight/internal/data"
	mockdb "github.com/djudju12/greenlight/internal/mocks"
	"github.com/djudju12/greenlight/internal/util"
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

	app := &application{
		models: data.Models{
			Users: mockdb.NewMockUserQuerier(ctrl),
		},
		mailer: mockdb.NewMockMailer(ctrl),
	}

	return test{
		app:      app,
		recorder: recorder,
		url:      url,
	}
}

type eqUserMatcher struct {
	user              data.User
	plainTextPassword string
}

func (eq eqUserMatcher) Matches(x any) bool {
	user, ok := x.(data.User)
	if !ok {
		return false
	}

	ok, err := user.Password.Matches(eq.plainTextPassword)
	if err != nil || !ok {
		return false
	}

	user.Password = eq.user.Password
	return reflect.DeepEqual(eq.user, user)
}

func (eq eqUserMatcher) String() string {
	return fmt.Sprintf("matchs arg %v and password %v", eq.user, eq.plainTextPassword)
}

func EqUser(user data.User, plainTextPassword string) gomock.Matcher {
	return eqUserMatcher{user, plainTextPassword}
}

func TestRegisterUserHandle(t *testing.T) {
	user, plaintextPassword := randomUser()

	testCases := []struct {
		name          string
		buildStubs    func(users *mockdb.MockUserQuerier, mailer *mockdb.MockMailer)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		requestBody   RegisterUserRequest
	}{
		{
			name: "User Register Handle - 200 OK",
			buildStubs: func(users *mockdb.MockUserQuerier, mailer *mockdb.MockMailer) {
				expectedUser := data.User{
					Name:      user.Name,
					Email:     user.Email,
					Activated: false,
				}

				users.EXPECT().
					Insert(EqUser(expectedUser, plaintextPassword)).
					Return(nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newTest(t, "/v1/movies")

			request := httptest.NewRequest(http.MethodPost, test.url, nil)

			// when
			test.app.registerUserHandle(test.recorder, request)

			// then

		})
	}
}

func randomUser() (data.User, string) {
	pwd := util.RandomPassword()
	user := data.User{
		Name:      util.RandomFullName(),
		Email:     util.RandomEmail(),
		Activated: true,
	}

	return user, pwd
}
