-- migrate:up

CREATE FUNCTION testdb() RETURNS text AS $$
  BEGIN
    RETURN 'dummy';
  END;
$$ LANGUAGE plpgsql;

-- migrate:down

DROP FUNCTION testdb;
