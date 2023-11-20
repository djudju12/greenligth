//go:build integration
// +build integration

package data

import (
	"testing"
	"time"

	"github.com/djudju12/greenlight/internal/util"
	"github.com/stretchr/testify/require"
)

func TestInsertMovies(t *testing.T) {
	movie := randomMovie()
	newMovie(t, &movie)
}

func TestGetAllMovie(t *testing.T) {
	n := 5
	var expectedMovies []*Movie
	movie := randomMovie()
	for i := 0; i < n; i++ {
		movieCopy := movie
		newMovie(t, &movieCopy)
		expectedMovies = append(expectedMovies, &movieCopy)
	}

	f := Filters{
		Page:         1,
		PageSize:     n,
		Sort:         "title",
		SortSafelist: []string{"title"},
	}

	actualMovies, _, err := testModels.Movies.GetAll(movie.Title, movie.Genres, f)

	require.NoError(t, err)

	require.Len(t, actualMovies, n)
	require.ElementsMatch(t, expectedMovies, actualMovies)
}

func TestDeleteMovie(t *testing.T) {
	movie := randomMovie()
	newMovie(t, &movie)

	err := testModels.Movies.Delete(movie.ID)
	require.NoError(t, err)

	_, err = testModels.Movies.Get(movie.ID)
	require.ErrorIs(t, ErrRecordNotFound, err)

	err = testModels.Movies.Delete(util.RandomInt(1000, 2000))
	require.ErrorIs(t, ErrRecordNotFound, err)
}

func TestGetMovie(t *testing.T) {
	expectedMovie := randomMovie()
	newMovie(t, &expectedMovie)

	actualMovie, err := testModels.Movies.Get(expectedMovie.ID)
	require.NoError(t, err)

	verifyMovies(t, *actualMovie, expectedMovie)

	_, err = testModels.Movies.Get(util.RandomInt(1000, 2000))
	require.ErrorIs(t, err, ErrRecordNotFound)
}

func TestUpdateMvoie(t *testing.T) {
	beforeMovie := randomMovie()
	newMovie(t, &beforeMovie)

	tempMovie := randomMovie()
	beforeMovie.Title = tempMovie.Title
	beforeMovie.Year = tempMovie.Year
	beforeMovie.Runtime = tempMovie.Runtime
	beforeMovie.Genres = tempMovie.Genres

	err := testModels.Movies.Update(&beforeMovie)
	require.NoError(t, err)

	afterMovie, err := testModels.Movies.Get(beforeMovie.ID)
	require.NoError(t, err)

	verifyMovies(t, beforeMovie, *afterMovie)

	afterMovie.Version = afterMovie.Version + 1
	err = testModels.Movies.Update(afterMovie)
	require.ErrorIs(t, err, ErrEditConflict)

	notExistingMovie := randomMovie()
	err = testModels.Movies.Update(&notExistingMovie)
	require.ErrorIs(t, err, ErrEditConflict)
}

func verifyMovies(t *testing.T, expected Movie, actual Movie) {
	require.Equal(t, actual.ID, expected.ID)
	require.Equal(t, actual.Title, expected.Title)
	require.Equal(t, actual.Year, expected.Year)
	require.Equal(t, actual.Runtime, expected.Runtime)
	require.Equal(t, actual.Version, expected.Version)
	require.Equal(t, actual.CreatedAt, expected.CreatedAt)
	require.ElementsMatch(t, actual.Genres, expected.Genres)
}

func newMovie(t *testing.T, movie *Movie) {
	err := testModels.Movies.Insert(movie)
	require.NoError(t, err)

	require.NotZero(t, movie.ID)
	require.NotZero(t, movie.Version)
	require.WithinDuration(t, time.Now(), movie.CreatedAt, time.Second)
}

func randomMovie() Movie {
	var genres []string
	for i := 0; i < 3; i++ {
		genres = append(genres, util.RandomString(10))
	}

	return Movie{
		Title:   util.RandomFullName(),
		Year:    int32(util.RandomInt(1900, 2023)),
		Runtime: Runtime(util.RandomInt(80, 240)),
		Genres:  genres,
	}
}
