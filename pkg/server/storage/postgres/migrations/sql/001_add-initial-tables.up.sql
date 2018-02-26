CREATE SCHEMA entity;

CREATE TABLE entity.patient (
  row_id SERIAL PRIMARY KEY,
  transaction_period TSTZRANGE NOT NULL DEFAULT tstzrange(NOW(), 'infinity', '[)'),
  entity_id VARCHAR UNIQUE NOT NULL,
  last_name VARCHAR,
  first_name VARCHAR,
  middle_name VARCHAR,
  suffix VARCHAR,
  birthdate DATE
);

CREATE TABLE entity.office (
  row_id SERIAL PRIMARY KEY,
  transaction_period TSTZRANGE NOT NULL DEFAULT tstzrange(NOW(), 'infinity', '[)'),
  entity_id VARCHAR UNIQUE NOT NULL,
  name VARCHAR
);
