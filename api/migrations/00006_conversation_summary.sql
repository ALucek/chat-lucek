-- +goose Up
alter table conversations add column summary text;
alter table conversations add column summary_through_message_id bigint
  references messages(id) on delete set null;

-- +goose Down
alter table conversations drop column summary_through_message_id;
alter table conversations drop column summary;
