create table bikes
(
    id       uuid  not null
        constraint bikes_id
            primary key,
    label    text  not null,
    imei     text  not null,
    location point not null
);

create unique index bikes_label
    on bikes (label);

create table stations
(
    id            uuid  not null
        constraint stations_id
            primary key,
    name          text  not null,
    address       text  not null,
    opening_hours text  not null,
    location      point not null,
    type          text  not null
);

create table customers
(
    id         uuid                                   not null
        constraint customers_pk
            primary key,
    auth0_id   text
        constraint customers_pk_3
            unique,
    stripe_id  text
        constraint customers_pk_2
            unique,
    created_at timestamp with time zone default now() not null
);

create table rides
(
    id                uuid                                                     not null
        constraint rides_pk
            primary key,
    bike_id           uuid                                                     not null,
    customer_id       uuid                                                     not null,
    started_at        timestamp with time zone                                 not null,
    ended_at          timestamp with time zone,
    charge_created_at timestamp with time zone,
    lock_user_id      bigint default random((0)::bigint, '4294967295'::bigint) not null
        constraint rides_pk_2
            unique
);

