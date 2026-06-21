-- +goose Up
create table users (
  id            bigserial primary key,
  email         text unique not null,
  password_hash text not null,
  created_at    timestamptz not null default now()
);

create table refresh_tokens (
  token      text primary key,
  user_id    bigint not null references users(id) on delete cascade,
  expires_at timestamptz not null,
  revoked    boolean not null default false,
  created_at timestamptz not null default now()
);

create table conversations (
  id         bigserial primary key,
  user_id    bigint not null references users(id) on delete cascade,
  title      text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table messages (
  id              bigserial primary key,
  conversation_id bigint not null references conversations(id) on delete cascade,
  role            text not null check (role in ('user','assistant')),
  content         text not null,
  created_at      timestamptz not null default now()
);

create index on conversations (user_id);
create index on messages (conversation_id);

-- +goose Down
drop table messages;
drop table conversations;
drop table refresh_tokens;
drop table users;