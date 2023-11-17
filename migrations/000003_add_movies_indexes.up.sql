CREATE INDEX IF NOT EXISTS movies_titles_idx ON movies USING GIN (to_tsvector('simple', title));

CREATE INDEX IF NOT EXISTS movies_genres_idx ON movies USING GIN (genres);