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
	"reflect"
	"testing"
	"time"

	"github.com/djudju12/greenlight/internal/data"
	"github.com/djudju12/greenlight/internal/jsonlog"
	mockdb "github.com/djudju12/greenlight/internal/mocks"
	"github.com/djudju12/greenlight/internal/util"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newMovieTest(t *testing.T, url string) test {
	ctrl := gomock.NewController(t)
	movies := mockdb.NewMockMovieQuerier(ctrl)

	recorder := httptest.NewRecorder()

	f, err := os.CreateTemp("", "tmpfile-")
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		models: &data.Models{
			Movies: movies,
		},
		logger: jsonlog.New(f, jsonlog.LevelInfo),
	}

	return test{
		recorder: recorder,
		url:      url,
		app:      app,
		tempFile: f,
	}
}

func TestCreateMovieHandler(t *testing.T) {
	expectedMovie := randomMovie()

	testCases := []struct {
		name          string
		requestBody   CreateMovieRequest
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name: "Create Movie Handler - 201 CREATED",
			requestBody: CreateMovieRequest{
				Title:   expectedMovie.Title,
				Year:    expectedMovie.Year,
				Runtime: expectedMovie.Runtime,
				Genres:  expectedMovie.Genres,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Insert(EqMovieRequest(expectedMovie)).
					DoAndReturn(func(movie *data.Movie) error {
						movie.ID = expectedMovie.ID
						movie.CreatedAt = expectedMovie.CreatedAt
						movie.Version = expectedMovie.Version
						return nil
					})

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, r.Code)
				requireHeaderHasEntry(t,
					r.Result().Header, "Location", fmt.Sprintf("/v1/movies/%d", expectedMovie.ID))

				requireBodyMatchMovie(t, r.Body, expectedMovie)
			},
		},
		{
			name: "Create Movie Handler - 500 DB RETURNED ERROR INSERTING MOVIE",
			requestBody: CreateMovieRequest{
				Title:   expectedMovie.Title,
				Year:    expectedMovie.Year,
				Runtime: expectedMovie.Runtime,
				Genres:  expectedMovie.Genres,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Insert(gomock.Any()).
					Return(errors.New("DB RETURNED ERROR"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "Create Movie Handler - 422 MOVIE WITH INVALID FIELDS",
			requestBody: CreateMovieRequest{
				Genres: expectedMovie.Genres,
			},
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no stubs for this tests")
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newMovieTest(t, "/v1/movies")

			tc.buildStubs(t, test.app)

			body, err := toReader(tc.requestBody)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, test.url, body)

			// when
			test.app.createMovieHandler(test.recorder, request)

			// then
			tc.checkResponse(t, test.recorder)
			test.close()
		})
	}
}

func TestUpdateMoviesHandler(t *testing.T) {
	movie := randomMovie()
	requestMovie := randomMovie()

	testCases := []struct {
		name          string
		requestBody   UpdateMovieRequest
		movieID       int64
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name:    "Test Update Movie Handler - 200 OK",
			movieID: movie.ID,
			requestBody: UpdateMovieRequest{
				Title:   &requestMovie.Title,
				Year:    &requestMovie.Year,
				Runtime: &requestMovie.Runtime,
				Genres:  requestMovie.Genres,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(movie.ID).
					Return(movie, nil)

				mockMovies.EXPECT().
					Update(gomock.Any()).
					Return(nil)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, r.Code)
				expectedMovie := movie
				expectedMovie.Title = requestMovie.Title
				expectedMovie.Year = requestMovie.Year
				expectedMovie.Runtime = requestMovie.Runtime
				expectedMovie.Genres = requestMovie.Genres
				requireBodyMatchMovie(t, r.Body, expectedMovie)
			},
		},
		{
			name:        "Test Update Movie Handler - 404 ID NOT FOUND",
			movieID:     movie.ID,
			requestBody: UpdateMovieRequest{},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(gomock.Any()).
					Return(&data.Movie{}, data.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, r.Code)
			},
		},
		{
			name: "Test Update Movie Handler - 404 REQUEST WITHOUT ID",
			// movieID:     movie.ID,
			requestBody: UpdateMovieRequest{},
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no stubs for this test")
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, r.Code)
			},
		},
		{
			name:        "Test Update Movie Handler - 500 DB RETURNED ERROR ON GETTING MOVIE",
			movieID:     movie.ID,
			requestBody: UpdateMovieRequest{},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(gomock.Any()).
					Return(&data.Movie{}, errors.New("DB RETURNED ERROR ON GETTING MOVIE"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name:        "Test Update Movie Handler - 409 DB RETURNED EDIT CONFLICT IN UPDATE MOVIE",
			movieID:     movie.ID,
			requestBody: UpdateMovieRequest{},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(movie.ID).
					Return(movie, nil)

				mockMovies.EXPECT().
					Update(gomock.Any()).
					Return(data.ErrEditConflict)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, r.Code)
			},
		},
		{
			name:        "Test Update Movie Handler - 500 DB RETURNED ERROR",
			movieID:     movie.ID,
			requestBody: UpdateMovieRequest{},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(movie.ID).
					Return(movie, nil)

				mockMovies.EXPECT().
					Update(gomock.Any()).
					Return(errors.New("DB RETURNED ERROR"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name:    "Test Update Movie Handler - 422 INVALID FIELD IN REQUEST",
			movieID: movie.ID,
			requestBody: UpdateMovieRequest{
				Genres: []string{"duplicated", "duplicated"},
			},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(movie.ID).
					Return(movie, nil)
''			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, r.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newMovieTest(t, fmt.Sprintf("/v1/movies/%d", tc.movieID))

			router := httprouter.New()
			router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", test.app.updatesMovieHandler)

			tc.buildStubs(t, test.app)

			body, err := toReader(tc.requestBody)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPatch, test.url, body)

			// when
			router.ServeHTTP(test.recorder, request)

			// then
			tc.checkResponse(t, test.recorder)
			test.close()
		})
	}
}

func randomMovie() *data.Movie {
	var genres []string
	for i := 0; i < 3; i++ {
		genres = append(genres, util.RandomString(10))
	}

	return &data.Movie{
		ID:        util.RandomInt(10, 1000),
		CreatedAt: time.Now(),
		Title:     util.RandomFullName(),
		Year:      int32(util.RandomInt(1900, 2023)),
		Runtime:   data.Runtime(util.RandomInt(80, 240)),
		Genres:    genres,
		Version:   1,
	}
}

func requireBodyMatchMovie(t *testing.T, body *bytes.Buffer, movie *data.Movie) {
	bytea, err := io.ReadAll(body)
	require.NoError(t, err)

	var envelope map[string]*data.Movie
	err = json.Unmarshal(bytea, &envelope)
	require.NoError(t, err)

	t.Logf("envelope %+v", envelope)

	gotMovie, exists := envelope["movie"]
	require.True(t, exists)
	require.NotNil(t, gotMovie)

	t.Logf("gotMovie %+v | movie %+v", gotMovie, movie)

	require.Equal(t, movie.ID, gotMovie.ID)
	require.Equal(t, movie.Title, gotMovie.Title)
	require.Equal(t, movie.Year, gotMovie.Year)
	require.Equal(t, movie.Runtime, gotMovie.Runtime)
	require.ElementsMatch(t, movie.Genres, gotMovie.Genres)
}

func requireHeaderHasEntry(t *testing.T, header http.Header, key string, values ...string) {
	results := header.Get(key)
	require.NotEmpty(t, results)

	t.Logf("header %+v | key %s | values %+v", header, key, values)
	require.Contains(t, values, results)
}

type eqRequestMovieMatcher struct {
	movie data.Movie
}

func (eq eqRequestMovieMatcher) Matches(x any) bool {
	movie, ok := x.(*data.Movie)
	if !ok {
		return false
	}

	if !reflect.DeepEqual(movie.Genres, eq.movie.Genres) {
		return false
	}

	return movie.Title == eq.movie.Title &&
		movie.Year == eq.movie.Year &&
		movie.Runtime == eq.movie.Runtime
}

func (eq eqRequestMovieMatcher) String() string {
	return fmt.Sprintf("match title, year, runtime, genres off %+v\n", eq.movie)
}

func EqMovieRequest(movie *data.Movie) gomock.Matcher {
	return eqRequestMovieMatcher{*movie}
}
