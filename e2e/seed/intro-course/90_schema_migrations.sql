--
-- Pins golang-migrate's bookkeeping table so the intro-course server's boot-time
-- `migrate ... up` is a no-op for the migrations already applied by the earlier
-- files in /docker-entrypoint-initdb.d.
--
-- The db container applies server/db/migration/0001..0006 and then
-- server/database_dumps/e2e_seed.sql (see docker-compose.e2e.yml), but migrate
-- itself never ran, so its table does not exist and the server would try to apply
-- 0001 again and fail on `CREATE TABLE developer_profile`.
--
-- Keep the version equal to the highest migration mounted by the compose file.
-- Future migrations (0007+) need no change here: the server applies them on top.
--
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL PRIMARY KEY,
    dirty boolean NOT NULL
);

INSERT INTO schema_migrations (version, dirty) VALUES (6, false);
