create table hostile (
  ts bigint not null,
  ip inet not null,
  cc char(2),
  asn text
);

alter table hostile add primary key (ts, ip);

create table country (
  a2 char(2) not null,
  name text not null
);

alter table country add primary key (a2);
