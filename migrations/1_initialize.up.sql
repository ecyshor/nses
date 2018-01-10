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
$$ LANGUAGE 'plpgsql';
CREATE TRIGGER update_modified_bill
  BEFORE UPDATE
  ON job_templates
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE TABLE jobs (

)