-- +goose Up
alter table messages add column langsmith_run_id text;

create table message_feedback (
  id                    bigserial primary key,
  message_id            bigint not null references messages(id) on delete cascade,
  user_id               bigint not null references users(id) on delete cascade,
  rating                smallint not null check (rating in (-1, 1)),
  comment               text,
  langsmith_feedback_id uuid not null,
  created_at            timestamptz not null default now(),
  updated_at            timestamptz not null default now(),
  unique (message_id, user_id)
);

-- +goose Down
drop table message_feedback;
alter table messages drop column langsmith_run_id;
