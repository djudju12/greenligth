package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/djudju12/greenlight/internal/validator"
	"github.com/lib/pq"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"-"`
}

var maxBytesTitle = 500

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= maxBytesTitle, "title", fmt.Sprintf("must not be more than %d bytes long", maxBytesTitle))

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")

	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

type MovieQuerier interface {
	Get(id int64) (*Movie, error)
	GetAll(title string, genres []string, f Filters) ([]*Movie, Metadata, error)
	Insert(movie *Movie) error
	Update(movie *Movie) error
	Delete(id int64) error
}

type MovieModel struct {
	DB *sql.DB
}

func (m MovieModel) Insert(movie *Movie) error {
	query := `
	INSERT INTO movies (title, year, runtime, genres)
	VALUES ($1, $2, $3, $4)
	RETURNING  id, created_at, version`

	args := []interface{}{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Get(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
	SELECT id, created_at, title, year, runtime, genres, version
	FROM movies
	WHERE id=$1`

	var movie Movie

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

func (m MovieModel) Update(movie *Movie) error {
	query := `
	UPDATE movies
	SET title=$1, year=$2, runtime=$3, genres=$4, version = version + 1
	WHERE id=$5 AND version = $6
	RETURNING version`

	args := []interface{}{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
	DELETE FROM movies
	WHERE id=$1`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m MovieModel) GetAll(title string, genres []string, f Filters) ([]*Movie, Metadata, error) {
	// This SQL query is designed so that each of the filters behaves like it is ‘optional’. For
	// example, the condition (LOWER(title) = LOWER($1) OR $1 = '') will evaluate as true if
	// the placeholder parameter $1 is a case-insensitive match for the movie title or the
	// placeholder parameter equals '' . So this filter condition will essentially be ‘skipped’ when
	// movie title being searched for is the empty string "" .

	// The (genres @> $2 OR $2 = '{}') condition works in the same way. The @> symbol is the
	// ‘contains’ operator for PostgreSQL arrays, and this condition will return true if all values in
	// the placeholder parameter $2 are contained in the database genres field or the placeholder
	// parameter contains an empty array.

	// query := `
	// SELECT id, created_at, title, year, runtime, genres, version
	// FROM movies
	// WHERE (LOWER(title)=LOWER($1) OR $1='')
	// AND (genres @> $2 OR $2='{}')
	// ORDER BY id`
	////////////////////////

	// The to_tsvector('simple', title) function takes a movie title and splits it into lexemes.
	// We specify the simple configuration, which means that the lexemes are just lowercase
	// versions of the words in the title†. For example, the movie title "The Breakfast Club"
	// would be split into the lexemes 'breakfast' 'club' 'the'

	// The plainto_tsquery('simple', $1) function takes a search value and turns it into a
	// formatted query term that PostgreSQL full-text search can understand. It normalizes thesearch value (again using the simple configuration), strips any special characters, and
	// inserts the and operator & between the words. As an example, the search value "The Club"
	// would result in the query term 'the' & 'club' .
	//
	// The @@ operator is the matches operator. In our statement we are using it to check whether
	// the generated query term matches the lexemes. To continue the example, the query term
	// 'the' & 'club' will match rows which contain both lexemes 'the' and 'club' .

	query := fmt.Sprintf(`
	SELECT count(*) OVER(), id, created_at, title, year, runtime, genres, version
	FROM movies
	WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
	AND (genres @> $2 OR $2 = '{}')
	ORDER BY %s %s, id ASC
	LIMIT $3 OFFSET $4`, f.sortColumn(), f.sortDirection())
	////////////////////////

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{
		title,            // $1
		pq.Array(genres), // $2
		f.limit(),        // $3
		f.offset(),       // $4
	}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	movies := []*Movie{}
	for rows.Next() {
		var movie Movie
		err = rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		movies = append(movies, &movie)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, f.Page, f.PageSize)
	return movies, metadata, nil
}
