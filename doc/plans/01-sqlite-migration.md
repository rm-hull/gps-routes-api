
# Plan: SQLite3 Migration for gps-routes-api

## TL;DR

Migrating from PostgreSQL to SQLite3 is **highly feasible** for this use case. The dataset size (10k-100k routes), low concurrency requirements, and feature maturity of modern SQLite extensions enable feature parity while dramatically simplifying deployment. This plan uses **Spatialite** for geographic queries, **FTS5** for full-text search, and **JSON** for array operations. Estimated effort: 80-120 developer hours over 4-6 weeks.

---

## Feasibility Assessment

### ✅ Fully Supported Features

| PostgreSQL Feature                         | SQLite Equivalent               | Confidence | Notes                                                |
| ------------------------------------------ | ------------------------------- | ---------- | ---------------------------------------------------- |
| Full-text search (TSVECTOR)                | FTS5 built-in                   | ✅ 100%    | Better ranking flexibility; no rewrite               |
| PostGIS spatial queries                    | Spatialite extension            | ✅ 95%     | ST_Within, ST_DWithin, KNN all available             |
| Array operations (terrain[], activities[]) | JSON1 (json_extract, json_each) | ✅ 90%     | Slight schema change; queries need translation       |
| Batch transactions                         | SQLite transactions             | ✅ 100%    | Native support; single-threaded model suits use case |
| Connection pooling                         | sqlite3 driver pooling          | ✅ 100%    | Works with CGo driver                                |
| Indexes (B-tree, GIN, GIST)                | B-tree + expression indexes     | ✅ 85%     | Fewer index types, but sufficient for workload       |
| Upserts (ON CONFLICT DO UPDATE)            | SQLite INSERT OR REPLACE        | ✅ 100%    | Native SQL support                                   |

### ⚠️ Known Limitations & Mitigations

| Limitation                                                         | Impact                         | Mitigation                                        |
| ------------------------------------------------------------------ | ------------------------------ | ------------------------------------------------- |
| **Concurrent writes**                                              | Only one writer at a time      | ✅ N/A - low concurrency requirement              |
| **Distance ranking with coordinate transformation** (ST_Transform) | No built-in coordinate systems | ✅ Pre-compute in app layer; use metered distance |
| **Array unnesting syntax**                                         | Different operator set         | ✅ Use json_each() + CROSS JOIN                   |
| **Full-text ranking tie-breaking**                                 | Different algorithms           | ✅ Minor UX difference; acceptable                |
| **CGo compilation**                                                | Requires C compiler (gcc)      | ✅ Pin sqlite3 version; document dependencies     |
| **Database file locking on network**                               | Should not serve over network  | ✅ Single-machine deployment only                 |

---

## Implementation Plan

### Phase 1: Feasibility Validation & POC (Week 1 | 5-8 days)

**Objective:** Verify core features work as expected with SQLite + extensions.

**Steps:**

1. **Set up SQLite + extensions test environment**
    - _(Depends on: none)_
    - Install `go-sqlite3` locally: `go get github.com/mattn/go-sqlite3`
    - Verify CGo compilation works: `go test -v github.com/mattn/go-sqlite3` (ensures gcc is available)
    - Document required system packages (gcc, sqlite3-dev on Linux)

2. **Create POC database with test schema**
    - _(Parallel with step 1)_
    - Create minimal SQLite schema mirroring [db/migrations/00001_routes.up.sql](db/migrations/00001_routes.up.sql)
    - Enable extensions: `PRAGMA load_extension('mod_spatialite'); PRAGMA load_extension('fts5');`
    - Load 100 sample route records from existing PostgreSQL dump

3. **POC: Validate FTS5 query equivalence**
    - _(Depends on step 2)_
    - Implement FTS5 virtual table for full-text search
    - Test weighted search (equivalent to PostgreSQL `setweight()` - achievable via column ranking)
    - Verify prefix matching (`"hello:*"` becomes simpler FTS5 syntax)
    - Compare result ranking between PG and SQLite for 10 test queries

4. **POC: Validate Spatialite geospatial queries**
    - _(Depends on step 2)_
    - Load coordinate data for test routes
    - Test **bounding box**: `ST_Within(_geoloc, ST_MakeEnvelope(...))`
    - Test **distance-based nearby**: `ST_DWithin(...)` with distance in meters
    - Test **KNN sorting**: `ORDER BY distance` using Spatialite's distance operator
    - Verify coordinate transformation (WGS84 ↔ Web Mercator) works as expected

5. **POC: Validate JSON array operations**
    - _(Depends on step 2)_
    - Create JSON array columns for `terrain`, `activities`, `facilities`, `points_of_interest`
    - Test array overlap equivalent: `json_extract(activities, '$[*]') LIKE '%keyword%'` vs. native JSON functions
    - Test unnesting for facet queries: `SELECT ... FROM routes, json_each(activities)` equivalent
    - Compare performance vs. PostgreSQL native arrays

6. **Document findings & decision**
    - _(Depends on steps 3-5)_
    - Create `/doc/SQLITE_MIGRATION_REPORT.md` with POC results
    - Decision gate: Proceed if all critical queries validate with <5% ranking differences

**Deliverables:**

- POC database file
- Test query results comparison document
- CGo compilation troubleshooting guide

---

### Phase 2: Schema Design & Migration Scripts (Week 2 | 5-7 days)

**Objective:** Design SQLite schema and create reversible migration strategy.

**Steps:**

1. **Design production SQLite schema**
    - _(Depends on Phase 1 completion)_
    - Adapt [db/migrations/00001_routes.up.sql](db/migrations/00001_routes.up.sql):
        - Replace `TEXT[]` columns with `JSON` type
        - Replace `TSVECTOR` computed column with FTS5 virtual table (separate)
        - Keep `GEOMETRY(Point, 4326)` geometry column, Spatialite will handle it
        - Keep all indexes, translate GIN → expression indexes where needed
    - Create [db/migrations/00002_sqlite_schema.sql](db/migrations/00002_sqlite_schema.sql) with full SQLite DDL
    - Include FTS5 table creation: `CREATE VIRTUAL TABLE routes_fts USING fts5(...)`

2. **Create bidirectional schema migration tool**
    - _(Depends on step 1)_
    - Write [cmds/migrate_postgres_to_sqlite.go](cmds/migrate_postgres_to_sqlite.go):
        - Connect to PostgreSQL source (connection string from env)
        - Fetch all route records in batches (1000 at a time)
        - Transform arrays → JSON, keep geometry as WKB
        - Insert into SQLite target database
        - Validate record counts match
        - Rollback capability (drop destination tables)
    - Include related tables: nearby, images, details

3. **Design concurrent migration with validation**
    - _(Parallel with step 2)_
    - Strategy: Run PostgreSQL & SQLite side-by-side during migration
    - Write [tests/migration_validation_test.go](tests/migration_validation_test.go):
        - Random sample query from both databases (FTS, spatial, faceted search)
        - Verify results match within acceptable difference (e.g., ranking order tolerance)
        - Fail if counts differ or essential records missing

4. **Create rollback/revert procedure**
    - _(Depends on steps 1-3)_
    - Document in [doc/SQLITE_MIGRATION_RUNBOOK.md](doc/SQLITE_MIGRATION_RUNBOOK.md):
        - Points-in-time for rollback (pre-migration, mid-migration, post-migration)
        - Automated rollback: keep PostgreSQL DB available for N days post-cutover
        - Data validation queries: count records, verify spatial integrity
        - Switchover playbook: update connection strings, restart services

**Deliverables:**

- [db/migrations/00002_sqlite_schema.sql](db/migrations/00002_sqlite_schema.sql)
- [cmds/migrate_postgres_to_sqlite.go](cmds/migrate_postgres_to_sqlite.go)
- [tests/migration_validation_test.go](tests/migration_validation_test.go)
- [doc/SQLITE_MIGRATION_RUNBOOK.md](doc/SQLITE_MIGRATION_RUNBOOK.md)

---

### Phase 3: Query Layer Refactoring (Week 3-4 | 10-12 days)

**Objective:** Replace PostgreSQL-specific SQL with SQLite equivalent while maintaining query builder API.

**Steps:**

1. **Create SQLite dialect abstraction**
    - _(Depends on none)_
    - Add [db/dialect.go](db/dialect.go) interface:
        ```go
        type Dialect interface {
            BuildArrayOverlapQuery(field string, values []string) string
            BuildFullTextQuery(query string) string
            BuildGeoQuery(operation string, params...interface{}) string
            FormatParam(index int) string  // $1 for PG, ? for SQLite
        }
        ```
    - Implement `PostgreSQLDialect` and `SQLiteDialect` structs
    - Register in query builder

2. **Port [db/query_builder.go](db/query_builder.go) to dialect-aware**
    - _(Depends on step 1)_
    - Add `dialect Dialect` field to `QueryBuilder` struct
    - Replace hardcoded `$1`, `$2` with `dialect.FormatParam(i)`
    - Replace array overlap `&&` operator:
        - PG: `activities && $1::TEXT[]`
        - SQLite: Use JSON array operations or materialized unnested columns (decision needed: see Further Considerations)
    - Replace FTS query operator `@@`:
        - Both use `fts_table MATCH query` (SQLite FTS5 syntax)
    - Update [db/query_builder_test.go](db/query_builder_test.go) with parameterized tests for both dialects

3. **Port spatial queries**
    - _(Depends on step 1)_
    - Update [db/query_builder.go](db/query_builder.go#L173-L189) geospatial filter building:
        - **Bounding box (ST_Within)**: Identical syntax in Spatialite, no changes needed
        - **Distance-based nearby (ST_DWithin)**: Verify Spatialite function signatures match
        - **KNN sorting**: Both support distance operator; update sort field construction
    - Create [db/spatial_test.go](db/spatial_test.go) with 10 test cases: bbox, distance, KNN, edge cases (null coords, boundary conditions)

4. **Port full-text search & ranking**
    - _(Depends on step 1)_
    - FTS5 uses different ranking function: `rank` instead of `ts_rank_cd`
    - Update sort field in [repositories/postgres.go](repositories/postgres.go#L318):
        - PG: `ts_rank_cd(search_vector, to_tsquery($1), 32) DESC`
        - SQLite: `rank DESC` (FTS5 default; or use bm25() for tuning)
    - Test weighted search (A=title, B=address, C=description) via FTS5 column creation options

5. **Port array unnesting for facets**
    - _(Depends on step 1)_
    - Update [services/facets.go](services/facets.go) facet query building:
        - PG: `SELECT UNNEST(terrain) AS key, COUNT(*)`
        - SQLite: `SELECT json_extract(terrain, '$[*]') AS key, COUNT(*)` (if JSON) OR use pre-normalized columns
        - Decide: (1) Keep JSON + json_each() in queries, (2) Materialize array elements in separate tables, or (3) Use SQLite columns for searchable arrays
        - Implement chosen approach; update facet query test cases

6. **Update batch insert logic for upserts**
    - _(Depends on step 1)_
    - Rewrite [repositories/postgres.go](repositories/postgres.go#L44-L78) `batch` operations:
        - PG: `INSERT ... ON CONFLICT (object_id) DO UPDATE`
        - SQLite: `INSERT OR REPLACE INTO routes (...) VALUES (...)` OR `INSERT INTO routes ... ON CONFLICT DO UPDATE SET ...`
        - Both support upsert; verify behavior matches (PG will update only specified columns; SQLite REPLACE replaces entire row)
        - Choose SQLite INSERT...ON CONFLICT if column-level control needed; REPLACE if full-row replacement is acceptable

**Deliverables:**

- [db/dialect.go](db/dialect.go) interface + implementations
- Updated [db/query_builder.go](db/query_builder.go) with dialect support
- Updated [db/query_builder_test.go](db/query_builder_test.go) parameterized tests
- [db/spatial_test.go](db/spatial_test.go)
- Updated [repositories/postgres.go](repositories/postgres.go)

---

### Phase 4: Repository Layer & Core Business Logic (Week 4-5 | 10-14 days)

**Objective:** Rewrite data access layer for SQLite while maintaining API contract.

**Steps:**

1. **Create SQLite repository implementation**
    - _(Depends on Phase 3 completion)_
    - Copy [repositories/postgres.go](repositories/postgres.go) → [repositories/sqlite.go](repositories/sqlite.go)
    - Update all query execution methods:
        - Replace `pgx.Conn.Query()` with `*sql.DB.QueryContext()`
        - Replace `pgx.Batch` with standard SQL transactions
        - Update connection pooling: use standard `sql.DB` pool (configured via `SetMaxOpenConns(25)`, `SetMaxIdleConns(5)`)
        - Remove pgx-specific error handling; use `database/sql` standard errors
    - Implement connection string parsing for SQLite (simpler: just file path):
        - `sqlite3:///path/to/routes.db?cache=shared&mode=rwc&_journal_mode=WAL`
        - WAL mode enables better concurrency for reads during writes

2. **Implement SQLite-specific transaction handling**
    - _(Depends on step 1)_
    - Update [repositories/sqlite.go](repositories/sqlite.go) `Store()` method:
        - Use `Begin(TxOptions{Isolation: LevelImmediate})` for immediate locking (prevents concurrent writes during transaction)
        - Batch inserts in `sql.Stmt` prepared statements
        - Handle `SQLITE_BUSY` errors with exponential backoff retry (for WAL contention)
    - Create [repositories/sqlite_test.go](repositories/sqlite_test.go) with concurrent write test (verify single-writer safety)

3. **Port connection management**
    - _(Depends on step 1)_
    - Update [db/postgres_connect.go](db/postgres_connect.go) → [db/sqlite_connect.go](db/sqlite_connect.go):
        - Load Spatialite extension on connection open: `db.Exec("SELECT load_extension('mod_spatialite')")`
        - Initialize FTS5: `db.Exec("SELECT load_extension('fts5')")` (usually built-in)
        - Set pragmas for performance & integrity:
            - `PRAGMA foreign_keys = ON` (enable cascade deletes)
            - `PRAGMA journal_mode = WAL` (write-ahead logging for concurrency)
            - `PRAGMA synchronous = NORMAL` (balance safety & performance)
    - Document in [README.md](README.md#SQLite-Setup) required extensions and build flags

4. **Update search & faceting logic**
    - _(Depends on steps 1-3)_
    - Port [repositories/postgres.go](repositories/postgres.go) `SearchHits()` method → [repositories/sqlite.go](repositories/sqlite.go):
        - Update WHERE clause builder for SQLite dialect (Step 3.5 above)
        - FTS query execution: `SELECT * FROM routes JOIN routes_fts ON routes.rowid = routes_fts.rowid WHERE routes_fts MATCH $1`
        - Spatial query execution: verify Spatialite functions work identically
    - Port [repositories/postgres.go](repositories/postgres.go) `FacetCounts()` method:
        - Execute 12 facet queries in parallel goroutines (same pattern as PostgreSQL version)
        - Handle array unnesting per decision in Phase 3.5

5. **Support legacy PostgreSQL builds (optional feature flag)**
    - _(Parallel with step 4, lower priority)_
    - Add build tag: `//go:build postgres` vs. `//go:build sqlite`
    - Maintain [repositories/postgres.go](repositories/postgres.go) as legacy path
    - Document in [CONTRIBUTING.md](CONTRIBUTING.md):
        - `go build -tags=sqlite` (default: SQLite)
        - `go build -tags=postgres` (legacy: PostgreSQL)

**Deliverables:**

- [repositories/sqlite.go](repositories/sqlite.go) with full data access layer
- [db/sqlite_connect.go](db/sqlite_connect.go)
- [repositories/sqlite_test.go](repositories/sqlite_test.go) with concurrent write safety tests
- Updated [db/sqlite_connect.go](db/sqlite_connect.go) with extension loading

---

### Phase 5: Testing, Performance Validation & Deployment (Week 5-6 | 10-12 days)

**Objective:** Validate feature parity, performance, and production readiness.

**Steps:**

1. **Create integration test suite**
    - _(Depends on Phase 4 completion)_
    - Refactor [tests/http_api_server_test.go](tests/http_api_server_test.go) to be database-agnostic:
        - Parameterize fixtures to run against both PostgreSQL and SQLite
        - Add 50+ test cases covering:
            - Full-text search (simple, prefix, multi-word, edge cases)
            - Geographic search (bbox, nearby, edge coordinates)
            - Faceted search (all 12 facets, combinations)
            - Pagination (offset, limit edge cases)
            - Upserts (insert, update, conflict handling)
            - Bulk operations (batch import of 1000+ records)
    - Target: >90% code coverage for [repositories/sqlite.go](repositories/sqlite.go)

2. **Performance benchmarking**
    - _(Depends on step 1)_
    - Create [tests/benchmark_sqlite_postgres_test.go](tests/benchmark_sqlite_postgres_test.go):
        - Load dataset: 50,000 routes with geographic & metadata variety
        - Benchmark queries:
            - **FTS**: 100 unique search queries, measure avg latency + p95/p99
            - **Spatial**: 100 bounding box queries, 100 nearby distance queries
            - **Faceted**: Full facet query suite (12 concurrent facets)
            - **Upsert**: Batch insert 10,000 new routes, measure throughput
        - Success criteria:
            - SQLite within 2x PostgreSQL latency for typical queries
            - Sub-100ms for 50k-dataset FTS/spatial queries (API target)
    - Document results in `/doc/PERFORMANCE_COMPARISON.md`

3. **Data migration dry-run in staging**
    - _(Parallel with step 2)_
    - Run [cmds/migrate_postgres_to_sqlite.go](cmds/migrate_postgres_to_sqlite.go) on production PostgreSQL dump
    - Verify migration validation tests pass (from Phase 2)
    - Measure migration time for full dataset (set expectations for cutover)
    - Document in runbook: "Staging migration took X minutes for Y records"

4. **Update Docker & deployment**
    - _(Depends on step 3)_
    - Update [Dockerfile](Dockerfile):
        - Add build stage: install gcc, sqlite3-dev for CGo compilation
        - Build flags: optimize for SQLite driver size
        - Final image: include sqlite3 driver code (no external library needed)
    - Update [docker-compose.yml](docker-compose.yml):
        - Remove PostgreSQL service
        - Add SQLite volume mount for persistence: `- ./data/routes.db:/app/data/routes.db`
    - Add health check: ping `/v1/gps-routes/ref-data` endpoint (validates FTS + spatial indexing)

5. **Update documentation**
    - _(Depends on steps 1-4)_
    - Update [README.md](README.md):
        - Database setup section: remove PostgreSQL instructions, add SQLite + Spatialite build requirements
        - Environment variables: remove PG\_\* vars, add `SQLITE_DB_PATH=/app/data/routes.db`
    - Create [doc/SQLITE_SETUP.md](doc/SQLITE_SETUP.md): system dependencies, CGo compiler setup, extension loading
    - Update [CONTRIBUTING.md](CONTRIBUTING.md): local dev setup with SQLite
    - Create [doc/SQLITE_ARCHITECTURE.md](doc/SQLITE_ARCHITECTURE.md): schema, query patterns, indexing strategy

6. **Caching layer compatibility check** _(optional)_
    - _(Depends on step 1)_
    - [repositories/cache.go](repositories/cache.go) should be DB-agnostic (already is)
    - Verify memoization works with SQLite queries: no changes needed
    - Consider: SQLite's built-in query caching; evaluate if additional application caching is still beneficial

**Deliverables:**

- [tests/http_api_server_test.go](tests/http_api_server_test.go) refactored (database-agnostic)
- 50+ new integration test cases
- [tests/benchmark_sqlite_postgres_test.go](tests/benchmark_sqlite_postgres_test.go)
- `/doc/PERFORMANCE_COMPARISON.md`
- Updated [Dockerfile](Dockerfile), [docker-compose.yml](docker-compose.yml)
- Updated [README.md](README.md), [CONTRIBUTING.md](CONTRIBUTING.md)
- New [doc/SQLITE_SETUP.md](doc/SQLITE_SETUP.md), [doc/SQLITE_ARCHITECTURE.md](doc/SQLITE_ARCHITECTURE.md)

---

## Relevant Files to Modify

### Critical Paths

- **[db/migrations/00002_sqlite_schema.sql](db/migrations/00002_sqlite_schema.sql)** ← NEW: SQLite schema definition
- **[db/migrations/00001_routes.up.sql](db/migrations/00001_routes.up.sql)** ← KEEP: Reference for schema design
- **[db/dialect.go](db/dialect.go)** ← NEW: Database dialect abstraction
- **[db/query_builder.go](db/query_builder.go)** ← MODIFY: Add dialect support
- **[db/sqlite_connect.go](db/sqlite_connect.go)** ← NEW: SQLite connection setup
- **[repositories/sqlite.go](repositories/sqlite.go)** ← NEW: SQLite data access layer
- **[repositories/postgres.go](repositories/postgres.go)** ← REFERENCE: Port logic from this
- **[db/postgres_connect.go](db/postgres_connect.go)** ← REFERENCE: Connection patterns

### Supporting Files

- **[cmds/migrate_postgres_to_sqlite.go](cmds/migrate_postgres_to_sqlite.go)** ← NEW: Schema migration tool
- **[cmds/http_api_server.go](cmds/http_api_server.go)** ← MODIFY: Update connection initialization
- **[Dockerfile](Dockerfile)** ← MODIFY: Add build dependencies
- **[docker-compose.yml](docker-compose.yml)** ← MODIFY: Remove PG, add SQLite volume
- **[main.go](main.go)** ← MODIFY: Register SQLite database driver

### Testing & Documentation

- **[tests/http_api_server_test.go](tests/http_api_server_test.go)** ← MODIFY: Parameterize for both DBs
- **[tests/benchmark_sqlite_postgres_test.go](tests/benchmark_sqlite_postgres_test.go)** ← NEW: Performance comparison
- **[tests/migration_validation_test.go](tests/migration_validation_test.go)** ← NEW: Migration correctness verification
- **[README.md](README.md)** ← MODIFY: Update setup instructions
- **[doc/SQLITE_MIGRATION_REPORT.md](doc/SQLITE_MIGRATION_REPORT.md)** ← NEW: POC findings
- **[doc/SQLITE_MIGRATION_RUNBOOK.md](doc/SQLITE_MIGRATION_RUNBOOK.md)** ← NEW: Cutover runbook
- **[doc/PERFORMANCE_COMPARISON.md](doc/PERFORMANCE_COMPARISON.md)** ← NEW: Benchmark results

---

## Verification Steps

### Phase 1 Verification

1. ✅ POC database created; schema loads without errors
2. ✅ 10 test FTS queries produce results; ranking order within 5% of PostgreSQL
3. ✅ 50 spatial test queries return same record counts ±1 (tolerance for precision)
4. ✅ Array unnesting produces facet counts matching PostgreSQL (within 1%)
5. ✅ No CGo compilation errors on macOS, Linux, Windows (CI/CD)

### Phase 2 Verification

1. ✅ Migration tool imports 50,000 test records in <5 minutes
2. ✅ Migration validation tests: all queries pass (FTS, spatial, facets)
3. ✅ Record count matches: PostgreSQL record count == SQLite record count (exact)
4. ✅ Rollback procedure tested: revert to PostgreSQL and re-run import

### Phase 3 Verification

1. ✅ Query builder tests pass for both PostgreSQL & SQLite dialects
2. ✅ All spatial test cases pass (10 test cases from [db/spatial_test.go](db/spatial_test.go))
3. ✅ Array query translation produces correct results (facet test suite)
4. ✅ Parameterized test suite runs against both databases (100+ test cases)

### Phase 4 Verification

1. ✅ SQLite repository implements all methods from PostgreSQL counterpart
2. ✅ Concurrent write test passes: multiple goroutines attempt writes; only one succeeds per transaction
3. ✅ Full API test suite passes against SQLite backend (+50 new test cases)
4. ✅ Transaction rollback works: failed writes leave database unchanged

### Phase 5 Verification

1. ✅ FTS benchmark: SQLite P99 latency <2x PostgreSQL (target <100ms)
2. ✅ Spatial benchmark: same latency targets met
3. ✅ Faceted search: 12-parallel-query execution time within budget
4. ✅ Upsert throughput: >1000 inserts/sec for new records
5. ✅ Docker image builds without errors; `docker-compose up` runs API successfully
6. ✅ Health check passes: `/v1/gps-routes/ref-data` returns 200 with valid JSON
7. ✅ API compatibility: all existing endpoint tests pass without modification

---

## Decisions & Design Rationale

### Array Handling: JSON vs. Materialized Columns

**Decision:** Use JSON1 extension with json_extract() + json_each() for query translation.

**Rationale:**

- **Minimal schema changes**: Keep existing query builder logic mostly intact
- **Flexibility**: Can easily add new array fields without schema migration
- **Performance trade-off acceptable**: Faceted search (array unnesting) is secondary feature; latency budget is generous
- **Alternative rejected**: Materialized lookup tables would require significant schema redesign and trigger complexity

### Connection Pooling Strategy

**Decision:** Use standard `sql.DB` with WAL mode (`pragma journal_mode=WAL`) for better read-write concurrency.

**Rationale:**

- **Single writer guarantee**: WAL mode serializes writes; aligns with low-concurrency requirement
- **Non-blocking reads**: Readers don't block writers (and vice versa) in WAL mode
- **Simple**: No need for external connection pool manager; stdlib handles it

### Upsert Strategy

**Decision:** Use `INSERT OR REPLACE` for full-row replacement (not column-selective UPDATE).

**Rationale:**

- **Simpler implementation**: Reduces migration effort from PostgreSQL `ON CONFLICT DO UPDATE`
- **Acceptable semantics**: Data import workflow replaces entire route objects anyway
- **Performance**: REPLACE is O(log n) like UPDATE; no meaningful difference

### Staged Rollout (Optional Future)

**Decision:** Maintain PostgreSQL repository layer as legacy code (`-tags=postgres`) for safe rollback.

**Rationale:**

- **Zero-downtime migration possible**: Run new SQLite backend in parallel; compare results before cutover
- **Rollback safety**: If issues emerge post-migration, revert binary to PostgreSQL version
- **Cost**: ~20-30 extra developer hours (Phase 4, step 5); defer unless cutover risk is high

### Extensibility: FTS Ranking Algorithm

**Decision:** Use SQLite FTS5 default ranking (bm25); can customize via rank() UDF in future.

**Rationale:**

- **Good enough for initial release**: FTS5 built-in ranking is tuned for general queries
- **Weighted search**: Column assignment weights (title, address, description) can be assigned at FTS table creation
- **Future optimization**: If search UX needs ranking tuning, implement custom rank() function (Phase 6)

---

## Known Risks & Mitigation

| Risk                                                      | Probability | Impact                               | Mitigation                                                                                    |
| --------------------------------------------------------- | ----------- | ------------------------------------ | --------------------------------------------------------------------------------------------- |
| **Spatialite distance queries not exact match PG**        | Medium      | API returns different nearby results | Phase 1 POC with 50 test queries; establish tolerance (e.g., ±10m)                            |
| **CGo build environment inconsistent**                    | Low         | Deployment fails on CI/CD            | Document build dependencies; pin go-sqlite3 version; provide Dockerfile with pre-built binary |
| **WAL mode file locking over network (if attempted)**     | Low         | Corruption or errors                 | Document: SQLite not suitable for network-mounted DBs; single-machine deployment only         |
| **FTS5 ranking produces significantly different results** | Medium      | Search UX regression                 | Phase 1 POC + Phase 5 user acceptance testing; establish ranking tolerance                    |
| **Faceted search latency increases 3x+**                  | Low         | API latency SLA miss                 | Phase 5 benchmarking; if needed, move facet computation to async background job               |
| **Data import tool bugs lose records**                    | Low         | Production data loss                 | Phase 2 validation tests (exact record count match) + Phase 3 dry-run on staging              |

**Mitigation overall:** POC (Phase 1) de-risks 80% of technical uncertainty; staged testing (Phase 3-5) catches integration issues early.

---

## Further Considerations

### 1. **Search Result Ranking Tolerance**

- **Question:** How much variance in full-text search result ranking is acceptable?
- **Impact:** Phase 1 POC ranking comparison determines if FTS5 is production-ready
- **Recommendation:** Establish 10-query test suite; acceptable if top-3 results match PostgreSQL for 80% of queries; if not, consider bm25 tuning or custom ranking UDF (Phase 6 effort)

### 2. **Array Query Performance vs. Simplicity**

- **Question:** Should array operations use JSON functions (slower, simpler) or materialized lookup tables (complex DDL, faster)?
- **Impact:** Faceted search latency; Phase 5 benchmarking determines if JSON performance is acceptable
- **Recommendation:** Start with JSON approach (Phase 3.5); if facet queries exceed 100ms SLA, refactor to lookup tables (Phase 6 optimization)

### 3. **Concurrent Write Handling & Queueing**

- **Question:** Should the app handle SQLite BUSY errors transparently (exponential backoff) or fail fast?
- **Impact:** User experience during concurrent imports; Phase 4 design choice
- **Recommendation:** Implement exponential backoff (3 retries, max 5-second delay) in repository layer; log/alert if QPS exceeds SQLite write capacity

### 4. **Spatialite Distance Units & Precision**

- **Question:** Does Spatialite's `ST_DWithin()` produce identical distance-meters behavior as PostgreSQL?
- **Impact:** Nearby route queries; Phase 1 POC must validate
- **Recommendation:** Test 50 nearby queries with various distances (1km, 10km, 50km); accept if results match within 10 meters

### 5. **API Versioning & Backward Compatibility**

- **Question:** Should migration be transparent to API clients, or is a new version/flag acceptable?
- **Impact:** Deployment complexity; client coordination
- **Recommendation:** Keep API contract identical (v1 endpoints unchanged); document database change in `/doc/ARCHITECTURE.md` (user-facing, not breaking)

---

## Timeline & Effort Estimate

| Phase                           | Duration      | Dev Days       | Risk Level | Blocker?              |
| ------------------------------- | ------------- | -------------- | ---------- | --------------------- |
| **1. POC & Feasibility**        | Week 1        | 5-8            | Medium     | 🔴 Yes—decision gate  |
| **2. Schema & Migration Tools** | Week 2        | 5-7            | Low        | 🟢 No                 |
| **3. Query Layer Refactoring**  | Week 3-4      | 10-12          | Medium     | 🟡 Depends on Phase 1 |
| **4. Repository & Core Logic**  | Week 4-5      | 10-14          | Low        | 🟡 Depends on Phase 3 |
| **5. Testing & Deployment**     | Week 5-6      | 10-12          | Low        | 🟢 No                 |
| **Reserve (unforeseen)**        | —             | 5-10           | —          | —                     |
| **TOTAL**                       | **4-6 weeks** | **45-63 days** | —          | —                     |

**Note:** Phases 2 and 3 can run in parallel after Phase 1 decision gate. Actual timeline depends on team size (1 dev vs. 2). Recommend: allocate 10-12 weeks for comfortable pacing + validation.

---

## Success Criteria (Go/No-Go Checklist)

- ✅ No breaking changes to REST API (100% backward compatible)
- ✅ FTS search results within acceptable ranking tolerance (Phase 1 gate)
- ✅ Geographic queries produce same nearby/bbox results (within precision tolerance)
- ✅ Faceted search queries complete in <100ms for 50k dataset
- ✅ Upsert throughput >1000 inserts/sec (data import performance maintained)
- ✅ Docker build succeeds on Linux, macOS, Windows (CI/CD)
- ✅ No data loss during migration (record count exact match)
- ✅ Rollback procedure tested in staging
- ✅ Load test: API survives 50 concurrent requests without deadlock or BUSY errors
- ✅ All existing tests pass; coverage >90% for new SQLite code

---

## Out of Scope (Phase 6+)

- Advanced FTS ranking customization (bm25 tuning, BM42 algorithm)
- Materialized array lookup tables (if JSON performance acceptable)
- Read replica / distributed SQLite (e.g., LiteFS)
- Query optimization beyond standard SQLite pragmas
- Alternative spatial libraries (h3geo, geohash)—stick with Spatialite

---

**Document Version:** 1.0 | **Date:** 12 April 2026 | **Status:** Ready for Phase 1 kickoff

---

## How to Use This Plan

1. **Phase 1 (Week 1):** Assign to senior backend engineer; execute POC; hold decision gate meeting
2. **Phase 2 (Week 2):** Parallel track: schema design + migration tooling
3. **Phase 3-4 (Weeks 3-5):** Core implementation; rotate team members for knowledge sharing
4. **Phase 5 (Weeks 5-6):** QA, performance validation, documentation
5. **Post-Phase 5:** Code review, staging deployment, production cutover (separate playbook)
