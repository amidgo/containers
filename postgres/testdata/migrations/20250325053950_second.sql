-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS animals (id serial not null, name varchar);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS animals;

-- +goose StatementEnd
