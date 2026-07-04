-- +goose Up
alter table messages add column trace jsonb;

-- +goose Down
alter table messages drop column trace;
