CREATE USER scheduler WITH PASSWORD 'scheduler';
CREATE DATABASE scheduler;
GRANT ALL PRIVILEGES ON DATABASE scheduler TO scheduler;

create table if not exists t_scheduler (
    id serial primary key ,
    action varchar(32) not null,
    payload  jsonb not null default '{}'::jsonb,
    state varchar(32) not null default 'SCHEDULED',
    result  jsonb not null default '{}'::jsonb,
    error  jsonb not null default '{}'::jsonb,
    attempts integer not null default 0,
    delayed_dt timestamp null default localtimestamp,
    created_dt timestamp not null default localtimestamp,
    updated_dt timestamp not null default localtimestamp
) WITH (
    autovacuum_vacuum_cost_delay=5, 
    autovacuum_vacuum_cost_limit=500,
    autovacuum_vacuum_scale_factor=0.0001,
    fillfactor=30
);

create unique index concurrently if not exists scheduler_object_index ON t_scheduler( (payload->'objectID'), action ) WITH (fillfactor=30);
create index concurrently task__state__delayed_dt__idx on t_scheduler (state, delayed_dt) WITH (fillfactor=30);

create table if not exists t_object (
    id serial primary key unique,
    data jsonb not null default '{}'::jsonb,
    created_dt timestamp not null default localtimestamp,
);

create table if not exists t_exported_object (
    id serial primary key unique,
    data jsonb not null default '{}'::jsonb,
    created_dt timestamp not null default localtimestamp,
);