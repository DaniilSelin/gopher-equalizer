CREATE SCHEMA IF NOT EXISTS %[1]s;

CREATE TABLE IF NOT EXISTS %[1]s.token_buckets (
    client_id    TEXT PRIMARY KEY,
    capacity     INTEGER NOT NULL,
    tokens       INTEGER NOT NULL,
    last_refill  TIMESTAMP WITH TIME ZONE NOT NULL

    CONSTRAINT ck_capacity_positive     CHECK (capacity > 0),
    CONSTRAINT ck_tokens_nonnegative    CHECK (tokens >= 0),
    CONSTRAINT ck_tokens_le_capacity    CHECK (tokens <= capacity)
);

-- Для алгоритмов, которые удаляют старых клинетов
CREATE INDEX IF NOT EXISTS idx_token_buckets_last_refill
  ON %[1]s.token_buckets (last_refill);

-- триггер, который обновляет last_refill, если меняются токены
-- Вместо того, чтобы в каждом запросе обнволять его самостоятельно
CREATE OR REPLACE FUNCTION %[1]s.update_last_refill()
  RETURNS trigger AS
$$
BEGIN
  NEW.last_refill := now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER update_last_refill_trigger
  BEFORE UPDATE ON %[1]s.token_buckets
  FOR EACH ROW
  WHEN (OLD.tokens IS DISTINCT FROM NEW.tokens)
  EXECUTE FUNCTION %[1]s.update_last_refill();