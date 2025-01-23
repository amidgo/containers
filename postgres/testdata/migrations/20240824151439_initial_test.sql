-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (id serial primary key, name varchar);

CREATE UNIQUE INDEX users_unique_name ON users (name);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
