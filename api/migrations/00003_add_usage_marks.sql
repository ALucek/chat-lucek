-- +goose Up
create table usage_marks (
  id           bigserial primary key,
  subject_hash bytea not null,
  created_at   timestamptz not null default now()
);
create index on usage_marks (subject_hash, created_at);

-- +goose Down
drop table usage_marks;
