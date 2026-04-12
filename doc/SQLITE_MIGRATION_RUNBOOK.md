# SQLite3 Migration Runbook

**Date:** 12 April 2026
**Procedure:** PostgreSQL → SQLite3 Schema & Data Migration
**Version:** 1.0

---

## Overview

This runbook provides step-by-step procedures for migrating from PostgreSQL to SQLite3, with rollback capabilities and validation checkpoints.

**Estimated Duration:** 30 minutes to 2 hours (depending on dataset size)
**Risk Level:** Medium (high confidence in migration tool, requires thorough staging test first)
**Rollback Window:** 7 days (requires keeping PostgreSQL DB available)

---

## Pre-Migration Checklist

### Phase 2.1: Preparation (Before Migration)

- [ ] **Backup PostgreSQL database**

    ```bash
    pg_dump -h localhost -U postgres routes > routes_backup_$(date +%Y%m%d_%H%M%S).sql
    ```

- [ ] **Verify migration tool**

    ```bash
    go build -o bin/migrate_postgres_to_sqlite ./cmds/migrate_postgres_to_sqlite.go
    bin/migrate_postgres_to_sqlite -h  # Print help
    ```

- [ ] **Test migration on staging environment** (MANDATORY)
    - Run migration on staging PostgreSQL copy
    - Execute validation tests
    - Measure migration time for full dataset
    - Document findings in migration log

- [ ] **Notify stakeholders**
    - Alert team of scheduled migration
    - Set maintenance window (e.g., low-traffic period)
    - Prepare communication for API downtime (if applicable)

- [ ] **Verify SQLite schema file exists**

    ```bash
    test -f db/migrations/00002_sqlite_schema.sql && echo "✅ Schema exists"
    ```

- [ ] **Check disk space on target machine**
    ```bash
    # Ensure SQLite DB file has space
    df -h /path/to/data/
    ```

---

## Migration Procedures

### Phase 2.2: Staging Migration (Test Run)

**Environment:** Staging/Development
**Duration:** 10-30 minutes (depending on data volume)

**Steps:**

1. **Create copy of PostgreSQL data (optional)**

    ```bash
    # Create a fresh DB copy for testing
    createdb -h localhost routes_migrate_test
    pg_dump -h localhost -U postgres routes | psql -h localhost -U postgres routes_migrate_test
    ```

2. **Run migration tool in dry-run mode**

    ```bash
    bin/migrate_postgres_to_sqlite \
      -pg-url "host=localhost user=postgres password=pwd dbname=routes_migrate_test" \
      -sqlite-db /tmp/routes_test.db \
      -dry-run
    ```

3. **Run migration with test data**

    ```bash
    bin/migrate_postgres_to_sqlite \
      -pg-url "host=localhost user=postgres password=pwd dbname=routes_migrate_test" \
      -sqlite-db /tmp/routes_test.db \
      -max-records 5000  # Test with subset
    ```

4. **Run validation tests**

    ```bash
    POSTGRESQL_URL="host=localhost user=postgres password=pwd dbname=routes_migrate_test" \
    go test -v ./tests -run TestMigrationValidation
    ```

5. **Check results**
    - Verify record counts match
    - Check FTS queries work
    - Verify geographic queries execute (without Spatialite: basic bbox only)
    - Confirm faceted counts match

6. **Document findings**
    - Migration time: **\_** minutes
    - Record count: **\_** routes
    - FTS validation: ✅ Pass / ❌ Fail
    - Facet validation: ✅ Pass / ❌ Fail
    - Issues found: ****\*\*****\_****\*\*****
    - Mitigation: ****\*\*****\_\_\_\_****\*\*****

7. **Cleanup staging test DB**
    ```bash
    dropdb -h localhost routes_migrate_test
    rm /tmp/routes_test.db
    ```

---

### Phase 2.3: Production Migration (Live)

**Environment:** Production
**Downtime:** ~10-30 minutes (API should be paused during migration)
**Backup:** PostgreSQL DB retained for 7 days

**Pre-Migration:**

1. **Notify users** (if applicable)
    - Post maintenance announcement
    - Set expected downtime window

2. **Verify PostgreSQL is accessible**

    ```bash
    psql -h <prod-db-host> -U <user> -d routes -c "SELECT COUNT(*) FROM routes;"
    ```

3. **Stop API service**

    ```bash
    systemctl stop gps-routes-api
    # OR: docker-compose stop api
    ```

4. **Wait for graceful shutdown** (~10 seconds)

**Migration:**

5. **Run migration tool**

    ```bash
    bin/migrate_postgres_to_sqlite \
      -pg-url "host=<prod-db-host> user=postgres password=*** dbname=routes" \
      -sqlite-db /data/routes.db
    ```

6. **Verify exit code**

    ```bash
    echo $?  # Should be 0 for success
    ```

7. **Check output log** for:
    - ✅ Connected to PostgreSQL
    - ✅ Routes migrated: **\_**
    - ✅ Nearby records migrated: **\_**
    - ✅ Images migrated: **\_**
    - ✅ Details migrated: **\_**
    - ✅ Migration validation passed

**Post-Migration:**

8. **Verify SQLite database**

    ```bash
    sqlite3 /data/routes.db "SELECT COUNT(*) FROM routes;"
    sqlite3 /data/routes.db ".tables"  # List all tables
    ```

9. **Check file permissions**

    ```bash
    ls -lh /data/routes.db
    chmod 644 /data/routes.db  # Readable by all, writable by owner
    ```

10. **Start API service with SQLite backend**

    ```bash
    # Update environment variable
    export SQLITE_DB_PATH=/data/routes.db
    systemctl start gps-routes-api
    # OR: docker-compose up -d
    ```

11. **Wait for service startup** (~5-10 seconds)

12. **Verify API is healthy**

    ```bash
    # Health check endpoint
    curl http://localhost:8080/v1/gps-routes/ref-data

    # Should return 200 with valid JSON facets
    ```

13. **Run smoke tests**
    - [ ] POST `/v1/gps-routes/search` with query `hiking` → returns results
    - [ ] GET `/v1/gps-routes/ref-data` → returns facets
    - [ ] GET `/v1/gps-routes/{objectID}` → returns specific route

14. **Monitor for errors**

    ```bash
    # Check application logs
    journalctl -u gps-routes-api -f  # 5 minutes of monitoring

    # OR: docker logs -f gps-routes-api
    ```

15. **Notify stakeholders**
    - Confirm migration successful
    - Resume normal operations

---

## Validation Procedures

### Phase 2.4: Post-Migration Validation

**Run immediately after migration:**

1. **Record count validation**

    ```bash
    sqlite3 /data/routes.db "SELECT COUNT(*) FROM routes;"
    # Compare with PostgreSQL count documented during staging
    ```

2. **Data integrity check**

    ```bash
    sqlite3 /data/routes.db << 'EOF'
    SELECT
      COUNT(*) as total_routes,
      COUNT(DISTINCT object_id) as unique_ids,
      COUNT(CASE WHEN latitude IS NULL THEN 1 END) as null_coords,
      COUNT(CASE WHEN activities IS NOT NULL THEN 1 END) as with_activities
    FROM routes;
    EOF
    ```

3. **Query performance check**

    ```bash
    # Full-text search (should be <100ms)
    time sqlite3 /data/routes.db "SELECT COUNT(*) FROM routes_fts WHERE routes_fts MATCH 'hiking'"

    # Bounding box (should be <100ms)
    time sqlite3 /data/routes.db "
      SELECT COUNT(*) FROM routes
      WHERE latitude BETWEEN 50 AND 58 AND longitude BETWEEN -5 AND -2
    "
    ```

4. **FTS validation**

    ```bash
    sqlite3 /data/routes.db "
    SELECT COUNT(*) as search_results
    FROM routes_fts
    WHERE routes_fts MATCH 'mountain*'
    "
    ```

5. **Array (JSON) validation**
    ```bash
    sqlite3 /data/routes.db "
    SELECT json_extract(activities, '$[0]') as first_activity
    FROM routes WHERE activities IS NOT NULL LIMIT 5
    "
    ```

---

## Rollback Procedures

### Phase 2.5: Rollback (If Migration Fails)

**If migration should fail during production, execute rollback within 30 minutes:**

**Option A: Revert to PostgreSQL (7-day window)**

1. **Stop the API**

    ```bash
    systemctl stop gps-routes-api
    ```

2. **Verify PostgreSQL is available**

    ```bash
    psql -h <prod-db-host> -U postgres -d routes -c "SELECT COUNT(*) FROM routes;"
    ```

3. **Update configuration to use PostgreSQL**

    ```bash
    # Change connection string in environment
    export POSTGRESQL_URL="host=<prod-db-host> user=postgres dbname=routes"
    unset SQLITE_DB_PATH
    ```

4. **Start API with PostgreSQL backend**

    ```bash
    systemctl start gps-routes-api
    ```

5. **Verify health**

    ```bash
    curl http://localhost:8080/v1/gps-routes/ref-data
    ```

6. **Investigate failure**
    - Review migration logs
    - Run staging tests to identify issue
    - Plan remediation

**Option B: Retry Migration (if schema issue)**

1. **Stop API and backup current SQLite DB**

    ```bash
    systemctl stop gps-routes-api
    mv /data/routes.db /data/routes.db.backup_$(date +%s)
    ```

2. **Fix identified schema issue** in `db/migrations/00002_sqlite_schema.sql`

3. **Re-run migration**

    ```bash
    bin/migrate_postgres_to_sqlite \
      -pg-url "host=<prod-db-host> user=postgres password=*** dbname=routes" \
      -sqlite-db /data/routes.db
    ```

4. **Restart API and verify**

**Option C: Restore from Backup (if data corruption)**

1. **Stop API**
2. **Restore PostgreSQL from backup**
    ```bash
    psql -h <prod-db-host> -U postgres -d routes < routes_backup_YYYYMMDD.sql
    ```
3. **Revert to PostgreSQL backend**
4. **Contact support for next steps**

---

## Monitoring & Maintenance

### Phase 2.6: Post-Migration Monitoring (7 days)

**Week 1 after migration:**

- [ ] **Daily log reviews**
    - Check for SQL errors in application logs
    - Monitor latency metrics
    - Track error rates

- [ ] **Weekly database health check**

    ```bash
    sqlite3 /data/routes.db "PRAGMA integrity_check;"
    # Should output: ok
    ```

- [ ] **Performance benchmarking**
    - Compare API latencies vs. PostgreSQL baseline
    - Track FTS query response times
    - Monitor database file size

- [ ] **Data validation (sample checks)**
    ```bash
    # Random route validation
    sqlite3 /data/routes.db "
    SELECT object_id, title, COUNT(DISTINCT json_each.value) as activity_count
    FROM routes, json_each(activities)
    GROUP BY object_id
    ORDER BY RANDOM() LIMIT 5
    "
    ```

### PostgreSQL Retention Policy

**During 7-day rollback window:**

- Keep PostgreSQL instance running (no writes)
- Backup location: `/backups/routes_postgresql_presqlite_migration.sql`
- After 7 days: Archive backup and decommission PostgreSQL (if migration successful)

---

## Troubleshooting

### Issue: Migration Hangs

**Symptoms:** Migration tool running for >1 hour without output
**Likely Cause:** Large dataset or slow database connection
**Fix:**

1. Check network connectivity to PostgreSQL
2. Monitor PostgreSQL server CPU/disk
3. Retry with `-max-records` limit to test smaller subset
4. Kill migration and investigate PostgreSQL server health

### Issue: Record Count Mismatch

**Symptoms:** SQLite count < PostgreSQL count
**Likely Cause:** Data insertion error (FK constraint, NULL handling)
**Fix:**

1. Check migration logs for error messages
2. Run validation test to identify which records failed
3. Verify foreign key relationships exist
4. Rollback and investigate schema definition

### Issue: FTS Queries Fail ("no such module: fts5")

**Symptoms:** FTS5 queries return "no such module" error
**Likely Cause:** FTS5 not compiled into SQLite driver
**Fix:**

1. Rebuild with flag: `CGO_ENABLED=1 go build -tags="sqlite_enable_fts5" -o app .`
2. Verify binary has FTS5: `strings bin/app | grep fts5`
3. See [SQLITE_CGO_BUILD_GUIDE.md](SQLITE_CGO_BUILD_GUIDE.md) for build details

### Issue: JSON Array Queries Return No Results

**Symptoms:** `json_each()` queries produce no rows
**Likely Cause:** JSON format conversion error from PostgreSQL arrays
**Fix:**

1. Check raw JSON values: `SELECT activities FROM routes LIMIT 1`
2. Verify format is valid JSON: `sqlite3 :memory: "SELECT json('[]')"`
3. If invalid, update `arrayToJSON()` function in migration tool
4. Re-run migration

---

## Logging & Audit Trail

All migration operations logged to:

- **Migration tool STDOUT:** `migration_$(date +%Y%m%d_%H%M%S).log`
- **Application logs:** `/var/log/gps-routes-api/` (or container logs)
- **Database audit:** Query logs (if enabled)

**Log retention:** 90 days

---

## Contingency Plans

### Scenario: Partial Migration Success

If only some tables migrate successfully:

1. **Do NOT start API** - will cause data inconsistency
2. **Run full rollback** to PostgreSQL backend
3. **Investigate** which tables failed
4. **Update migration tool** to handle edge cases
5. **Re-test** on staging
6. **Retry** full migration

### Scenario: Performance Degradation Post-Migration

If API latency increases significantly:

1. **Check indexes**: `PRAGMA index_info(idx_name);`
2. **Analyze query plan**: `EXPLAIN QUERY PLAN SELECT ...`
3. **Update statistics**: `PRAGMA analyze;`
4. **Review PRAGMA settings**: Check `cache_size`, `synchronous`, etc.
5. **Compare with PostgreSQL baseline**
6. **Optimize if needed** or fallback to PostgreSQL

---

## Post-Rollback Cleanup (If Aborted)

If rollback occurs and PostgreSQL is restored:

1. **Archive failed SQLite DB**

    ```bash
    tar -czf /backups/routes_sqlite_migration_failed_$(date +%Y%m%d).tar.gz /data/routes.db
    ```

2. **Remove migration artifacts**

    ```bash
    rm -f /tmp/routes_test.db*
    rm -f /data/routes.db
    ```

3. **Document findings** for post-mortem analysis

4. **Schedule retry** after fixing issues identified

---

## Sign-off

**Migration Executed By:** ****\*\*****\_****\*\***** (Date: **\_\_\_**)
**Verified By:** ****\*\*****\_****\*\***** (Date: **\_\_\_**)
**Status:** ☐ Success ☐ Rollback ☐ In-Progress

**Notes:**

---

---

---

**Document Version:** 1.0
**Last Updated:** 12 April 2026
**Next Review:** After Phase 2 staging test
