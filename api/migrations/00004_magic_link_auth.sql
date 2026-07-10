-- +goose Up
alter table users add constraint users_email_key unique (email);
alter table users drop column google_sub;

create table magic_links (
  token_hash text primary key,
  email      text not null,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

-- +goose Down
drop table magic_links;
alter table users add column google_sub text;
alter table users drop constraint users_email_key;
