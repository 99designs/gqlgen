-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TABLE IF NOT EXISTS Posts (
    post_id  BIGSERIAL  PRIMARY KEY,
    author_id UUID NOT NULL,
    title TEXT NOT NULL,
    text TEXT NOT NULL,
    comments_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    create_date TIMESTAMP  WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS Comments (
    comment_id BIGSERIAL  PRIMARY KEY,
    author_id UUID NOT NULL,
    post_id BIGINT NOT NULL REFERENCES posts(post_id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES comments(comment_id) ON DELETE CASCADE,
    path LTREE UNIQUE NOT NULL,
    replies_level INTEGER NOT NULL,
    text TEXT NOT NULL,
    create_date TIMESTAMP  WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX path_gist_idx ON Comments USING GIST (path);

CREATE INDEX create_date_idx ON Comments (create_date);

CREATE INDEX post_idx ON Comments (post_id);

-- +goose StatementEnd