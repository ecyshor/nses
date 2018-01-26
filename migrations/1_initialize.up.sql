CREATE EXTENSION "uuid-ossp";

CREATE TABLE job_templates (
  id         UUID      DEFAULT uuid_generate_v4() PRIMARY KEY,
  created    TIMESTAMP DEFAULT current_timestamp,
  modified   TIMESTAMP DEFAULT current_timestamp,
  job_type   VARCHAR(256) NOT NULL,
  properties JSONB
);


CREATE OR REPLACE FUNCTION update_modified_column()
  RETURNS TRIGGER AS $$
BEGIN
  IF ROW (NEW.*) IS DISTINCT FROM ROW (OLD.*)
  THEN
    NEW.modified = now();
    RETURN NEW;
  ELSE
    RETURN OLD;
  END IF;
END;
$$
LANGUAGE 'plpgsql';
CREATE TRIGGER update_modified_job_template
  BEFORE UPDATE
  ON job_templates
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE TABLE jobs (
  id                UUID      DEFAULT uuid_generate_v4() PRIMARY KEY,
  template          UUID REFERENCES job_templates,
  interval          VARCHAR(64) NOT NULL,
  created           TIMESTAMP DEFAULT current_timestamp,
  modified          TIMESTAMP DEFAULT current_timestamp,
  payload           JSONB,
  next_run_min_date TIMESTAMP,
  next_run_max_date TIMESTAMP
);

CREATE TRIGGER update_modified_job
  BEFORE UPDATE
  ON jobs
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE TABLE job_runs (
  id            UUID    DEFAULT uuid_generate_v4() PRIMARY KEY,
  run_date      TIMESTAMP,
  successfull   BOOLEAN DEFAULT TRUE,
  extra_details JSONB,
  job_id        UUID REFERENCES jobs
)