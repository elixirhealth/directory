
CREATE EXTENSION pg_trgm;

CREATE INDEX patient_name
ON entity.patient
USING GIN ((COALESCE(UPPER(last_name), '') || ' ' || COALESCE(UPPER(first_name), '')) gin_trgm_ops);

CREATE INDEX office_name
ON entity.office
USING GIN (COALESCE(UPPER(name),'') gin_trgm_ops);
