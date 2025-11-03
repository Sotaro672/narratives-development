
CREATE TABLE IF NOT EXISTS companies (
  id UUID PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  admin UUID NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by UUID NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by UUID NOT NULL,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by UUID NULL,
  CONSTRAINT fk_companies_admin
      FOREIGN KEY (admin)
      REFERENCES members(id)
      ON UPDATE CASCADE
      ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS company_update_logs (
  log_id BIGSERIAL PRIMARY KEY,
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  changed_fields JSONB NOT NULL,
  updated_by UUID,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('CREATE','UPDATE','DELETE'))
);

CREATE OR REPLACE FUNCTION trg_companies_update()
RETURNS TRIGGER AS $$
DECLARE
  diff JSONB := '{}'::jsonb;
BEGIN
  IF NEW.name IS DISTINCT FROM OLD.name THEN
    diff := jsonb_set(diff, '{name}', jsonb_build_object('old', OLD.name, 'new', NEW.name));
  END IF;
  IF NEW.admin IS DISTINCT FROM OLD.admin THEN
    diff := jsonb_set(diff, '{admin}', jsonb_build_object('old', OLD.admin, 'new', NEW.admin));
  END IF;
  IF NEW.is_active IS DISTINCT FROM OLD.is_active THEN
    diff := jsonb_set(diff, '{is_active}', jsonb_build_object('old', OLD.is_active, 'new', NEW.is_active));
  END IF;

  NEW.updated_at := NOW();
  NEW.updated_by := COALESCE(current_setting('app.user_id', true)::uuid, NEW.updated_by);

  INSERT INTO company_update_logs(company_id, changed_fields, updated_by, updated_at, operation_type)
  VALUES (OLD.id, diff, NEW.updated_by, NEW.updated_at, 'UPDATE');

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER companies_update_trg
BEFORE UPDATE ON companies
FOR EACH ROW
WHEN (OLD IS DISTINCT FROM NEW)
EXECUTE FUNCTION trg_companies_update();

CREATE OR REPLACE FUNCTION trg_companies_insert()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO company_update_logs(company_id, changed_fields, updated_by, updated_at, operation_type)
  VALUES (NEW.id, '{}'::jsonb, NEW.created_by, NEW.created_at, 'CREATE');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER companies_insert_trg
AFTER INSERT ON companies
FOR EACH ROW
EXECUTE FUNCTION trg_companies_insert();
