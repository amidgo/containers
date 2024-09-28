-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id serial primary key,
    name varchar
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
