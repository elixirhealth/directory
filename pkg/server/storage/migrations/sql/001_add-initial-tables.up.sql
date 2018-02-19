CREATE SCHEMA entity;

CREATE TABLE entity.patient (
  row_id SERIAL PRIMARY KEY,
  entity_id VARCHAR,
  last_name VARCHAR,
  first_name VARCHAR,
  middle_name VARCHAR,
  suffix VARCHAR,
  birthdate DATE
);

CREATE UNIQUE INDEX patient_entity_id ON entity.patient (entity_id);

CREATE TABLE entity.office (
  row_id SERIAL PRIMARY KEY,
  entity_id VARCHAR,
  name VARCHAR
);

CREATE UNIQUE INDEX office_entity_id ON entity.office (entity_id);
