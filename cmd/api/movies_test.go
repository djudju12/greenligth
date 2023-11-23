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
	"strings"
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
	movie := randomMovie()

	testCases := []struct {
		name          string
		requestBody   CreateMovieRequest
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name: "Create Movie Handler - 201 CREATED",
			requestBody: CreateMovieRequest{
				Title:   movie.Title,
				Year:    movie.Year,
				Runtime: movie.Runtime,
				Genres:  movie.Genres,
			},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				expectedMovie := &data.Movie{
					Title:   movie.Title,
					Year:    movie.Year,
					Genres:  movie.Genres,
					Runtime: movie.Runtime,
				}

				mockMovies.EXPECT().
					Insert(EqMovieRequest(expectedMovie)).
					DoAndReturn(func(m *data.Movie) error {
						m.ID = movie.ID
						m.CreatedAt = movie.CreatedAt
						m.Version = movie.Version
						return nil
					})

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, r.Code)
				requireHeaderHasEntry(t,
					r.Result().Header, "Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

				requireBodyMatchMovie(t, r.Body, movie)
			},
		},
		{
			name: "Create Movie Handler - 500 DB RETURNED ERROR INSERTING MOVIE",
			requestBody: CreateMovieRequest{
				Title:   movie.Title,
				Year:    movie.Year,
				Runtime: movie.Runtime,
				Genres:  movie.Genres,
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
				Genres: movie.Genres,
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
			},
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

func TestDeleteMovieHandler(t *testing.T) {
	movie := randomMovie()

	testCases := []struct {
		name          string
		movieID       int64
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name:    "Test Delete Movie Handler - 200 OK",
			movieID: movie.ID,
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Delete(movie.ID).
					Return(nil)

			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, r.Code)
			},
		},
		{
			name: "Test Delete Movie Handler - 404 NO ID PROVIDED",
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no stubs to build in this test")
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, r.Code)
			},
		},
		{
			name:    "Test Delete Movie Handler - 404 DB RETURN RECORD NOT FOUND",
			movieID: movie.ID,
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Delete(gomock.Any()).
					Return(data.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, r.Code)
			},
		},
		{
			name:    "Test Delete Movie Handler - 500 DB RETURN ERROR",
			movieID: movie.ID,
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Delete(gomock.Any()).
					Return(errors.New("DB ERROR"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newMovieTest(t, fmt.Sprintf("/v1/movies/%d", tc.movieID))

			router := httprouter.New()
			router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", test.app.deleteMovieHandler)

			tc.buildStubs(t, test.app)

			request := httptest.NewRequest(http.MethodDelete, test.url, nil)

			// when
			router.ServeHTTP(test.recorder, request)

			// then
			tc.checkResponse(t, test.recorder)
			test.close()
		})
	}
}

func TestShowMovieHandler(t *testing.T) {
	movie := randomMovie()

	testCases := []struct {
		name          string
		movieID       int64
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name:    "Test Show Movie Handler - 200 OK",
			movieID: movie.ID,
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(movie.ID).
					Return(movie, nil)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, r.Code)
				requireBodyMatchMovie(t, r.Body, movie)
			},
		},
		{
			name:    "Test Show Movie Handler - 404 DB RETURN RECORD NOT FOUND",
			movieID: movie.ID,
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(movie.ID).
					Return(&data.Movie{}, data.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, r.Code)
			},
		},
		{
			name:    "Test Show Movie Handler - 500 DB RETURN ERROR",
			movieID: movie.ID,
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				mockMovies.EXPECT().
					Get(movie.ID).
					Return(&data.Movie{}, errors.New("DB ERROR"))
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, r.Code)
			},
		},
		{
			name: "Test Show Movie Handler - 404 NO ID PROVIDED",
			buildStubs: func(t *testing.T, app *application) {
				t.Log("no need for stubs in this test")
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, r.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newMovieTest(t, fmt.Sprintf("/v1/movies/%d", tc.movieID))

			router := httprouter.New()
			router.HandlerFunc(http.MethodGet, "/v1/movies/:id", test.app.showMovieHandler)

			tc.buildStubs(t, test.app)

			request := httptest.NewRequest(http.MethodGet, test.url, nil)

			// when
			router.ServeHTTP(test.recorder, request)

			// then
			tc.checkResponse(t, test.recorder)
			test.close()
		})
	}
}

func TestListMoviesHandler(t *testing.T) {
	sortSafelist := []string{
		"id",
		"title",
		"year",
		"runtime",
		"-id",
		"-title",
		"-year",
		"-runtime",
	}

	var movies []*data.Movie
	n := 5
	for i := 0; i < n; i++ {
		movies = append(movies, randomMovie())
	}

	testCases := []struct {
		name          string
		requestParams ListMoviesRequest
		buildStubs    func(t *testing.T, app *application)
		checkResponse func(t *testing.T, r *httptest.ResponseRecorder)
	}{
		{
			name: "Test List Movie Handler - 200 OK",
			requestParams: ListMoviesRequest{
				Filters: data.Filters{
					Page:     1,
					PageSize: n,
					Sort:     "id",
				},
			},
			buildStubs: func(t *testing.T, app *application) {
				mockMovies, ok := app.models.Movies.(*mockdb.MockMovieQuerier)
				require.True(t, ok)

				expectedInput := ListMoviesRequest{
					Title:  "",
					Genres: []string{},
					Filters: data.Filters{
						Page:         1,
						PageSize:     n,
						Sort:         "id",
						SortSafelist: sortSafelist,
					},
				}

				mockMovies.EXPECT().
					GetAll(expectedInput.Title, expectedInput.Genres, expectedInput.Filters).
					Return(movies, data.Metadata{}, nil)
			},
			checkResponse: func(t *testing.T, r *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, r.Code)
				requireBodyMatchListMovies(t, r.Body, movies)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			test := newMovieTest(t, urlFromRequest("/v1/movies", tc.requestParams))

			t.Logf("url: %s", test.url)
			router := httprouter.New()
			router.HandlerFunc(http.MethodGet, "/v1/movies", test.app.listMoviesHandles)

			tc.buildStubs(t, test.app)

			request := httptest.NewRequest(http.MethodGet, test.url, nil)

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

func requireBodyMatchListMovies(t *testing.T, body *bytes.Buffer, movies []*data.Movie) {
	bytea, err := io.ReadAll(body)
	require.NoError(t, err)

	var envelope struct {
		Movies   []*data.Movie `json:"movies"`
		Metadata data.Metadata `json:"metadata"`
	}

	err = json.Unmarshal(bytea, &envelope)
	require.NoError(t, err)

	require.Equal(t, len(movies), len(envelope.Movies))
	require.NotNil(t, envelope.Metadata)

	for i, movie := range movies {
		requireMovieMatch(t, movie, envelope.Movies[i])
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

	requireMovieMatch(t, movie, gotMovie)
}

func requireMovieMatch(t *testing.T, expected *data.Movie, actual *data.Movie) {
	t.Logf("expected %+v | actual %+v", expected, actual)
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Title, actual.Title)
	require.Equal(t, expected.Year, actual.Year)
	require.Equal(t, expected.Runtime, actual.Runtime)
	require.ElementsMatch(t, expected.Genres, actual.Genres)
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

func urlFromRequest(baseUrl string, request ListMoviesRequest) string {
	url := fmt.Sprintf("%s?", baseUrl)

	if request.Title != "" {
		url = fmt.Sprintf("%stitle=%s&", url, request.Title)
	}

	if len(request.Genres) > 0 {
		url = fmt.Sprintf("%sgenres=%s&", url, strings.Join(request.Genres, ","))
	}

	if request.Page > 0 {
		url = fmt.Sprintf("%spage=%d&", url, request.Page)
	}

	if request.PageSize > 0 {
		url = fmt.Sprintf("%spage_size=%d&", url, request.PageSize)
	}

	if request.Sort != "" {
		url = fmt.Sprintf("%ssort=%s&", url, request.Sort)
	}

	return url[:len(url)-1]
}
