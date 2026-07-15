-- +goose Up
alter table conversations add column summary text;
alter table conversations add column summary_through_message_id bigint;
-- NOT VALID skips the blocking scan; the new column is all-null.
alter table conversations
  add constraint conversations_summary_through_message_id_fkey
  foreign key (summary_through_message_id) references messages(id)
  on delete set null not valid;

-- +goose Down
alter table conversations drop column summary_through_message_id;
alter table conversations drop column summary;
