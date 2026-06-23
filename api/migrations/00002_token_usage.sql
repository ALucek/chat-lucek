-- +goose Up
create table token_usage (
  id                bigserial primary key,
  user_id           bigint not null references users(id) on delete cascade,
  prompt_tokens     int not null,
  completion_tokens int not null,
  created_at        timestamptz not null default now()
);
create index on token_usage (user_id, created_at);

-- +goose Down
drop table token_usage;