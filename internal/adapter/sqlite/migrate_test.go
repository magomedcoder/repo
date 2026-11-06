package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/migrations"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/migrate"
)

func TestMigrateFreshDatabase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	var count int64
	if err := db.Model(&migrate.Record{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if int(count) != len(migrations.All()) {
		t.Fatalf("journal rows=%d want %d (%v)", count, len(migrations.All()), migrate.Names(migrations.All()))
	}

	ok, err := migrate.ColumnExists(db, "users", "role")
	if err != nil || !ok {
		t.Fatalf("users.role missing: %v", err)
	}

	ok, err = migrate.ColumnExists(db, "repositories", "issues_enabled")
	if err != nil || !ok {
		t.Fatalf("repositories.issues_enabled missing: %v", err)
	}

	store := NewRepositoryRepo(db)
	repo, err := store.FindByRepoPath(context.Background(), "missing.git")
	if err == nil || repo != nil {
		t.Fatal("expected missing repository")
	}

	if _, err := NewDB(dir); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateIdempotentOnExistingDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")

	db, err := openRawDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := bootstrapSchema(db); err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&migrate.Record{}); err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	u := &domain.User{
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "x",
		Role:         domain.RoleUser,
	}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	db2, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	sqlDB2, err := db2.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB2.Close()

	got, err := NewUserRepo(db2).FindByID(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Username != "bob" || got.Role != domain.RoleUser {
		t.Fatalf("user not preserved: %+v", got)
	}
}

func TestMigrateV1FromProdSchema(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")
	db, err := openRawDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, stmt := range []string{
		`CREATE TABLE schema_migrations (version integer PRIMARY KEY AUTOINCREMENT, name text NOT NULL, applied_at datetime NOT NULL)`,
		`CREATE TABLE users (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, updated_at datetime, deleted_at datetime, username text NOT NULL, email text NOT NULL, password_hash text NOT NULL, avatar_path text, role integer NOT NULL DEFAULT 0)`,
		`CREATE TABLE sessions (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, token text NOT NULL, user_id integer NOT NULL, expires_at datetime, last_seen_at datetime)`,
		`CREATE TABLE access_tokens (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, user_id integer NOT NULL, name text NOT NULL, token_hash text NOT NULL, prefix text NOT NULL, last_used_at datetime)`,
		`CREATE TABLE folders (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at datetime, updated_at datetime, deleted_at datetime,
			owner_kind text NOT NULL DEFAULT "user", owner_id integer NOT NULL,
			parent_id integer, name text NOT NULL, slug text NOT NULL, path text NOT NULL,
			CONSTRAINT fk_folders_parent FOREIGN KEY (parent_id) REFERENCES folders(id)
		)`,
		`CREATE UNIQUE INDEX idx_folder_scope ON folders (owner_kind, owner_id, ifnull(parent_id, 0), slug) WHERE deleted_at IS NULL`,
		`CREATE TABLE repositories (
			id integer PRIMARY KEY AUTOINCREMENT,
			created_at datetime, updated_at datetime, deleted_at datetime,
			name text NOT NULL, owner_kind text NOT NULL DEFAULT "user", owner_id integer NOT NULL,
			folder_id integer, description text, is_private numeric NOT NULL DEFAULT false,
			show_on_home numeric NOT NULL DEFAULT false, protect_default_branch numeric NOT NULL DEFAULT true,
			issues_enabled numeric NOT NULL DEFAULT true, default_branch text NOT NULL DEFAULT "main",
			path text NOT NULL, repo_path text NOT NULL DEFAULT "", last_activity_at datetime,
			CONSTRAINT fk_repositories_folder FOREIGN KEY (folder_id) REFERENCES folders(id)
		)`,
		`CREATE TABLE issues (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, updated_at datetime, repository_id integer NOT NULL, number integer NOT NULL, title text NOT NULL, body text, state text NOT NULL, author_id integer NOT NULL)`,
		`CREATE TABLE pull_requests (id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, updated_at datetime, repository_id integer NOT NULL, number integer NOT NULL, title text NOT NULL, body text, state text NOT NULL, author_id integer NOT NULL, base_branch text NOT NULL, head_branch text NOT NULL, merged_at datetime, merged_by integer)`,
		`INSERT INTO folders (owner_id, parent_id, name, slug, path) VALUES (1, NULL, 'work', 'work', 'org/work')`,
		`INSERT INTO repositories (name, owner_id, folder_id, path, repo_path) VALUES ('api', 1, 1, '/tmp/api.git', 'org/work/api.git')`,
		`INSERT INTO issues (repository_id, number, title, state, author_id) VALUES (1, 3, 't', 'open', 1)`,
		`INSERT INTO pull_requests (repository_id, number, title, state, author_id, base_branch, head_branch) VALUES (1, 2, 'p', 'open', 1, 'main', 'feat')`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	db2, err := NewDB(dir)
	if err != nil {
		t.Fatalf("migrate v1: %v", err)
	}
	sqlDB2, err := db2.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB2.Close()

	ok, err := migrate.ColumnExists(db2, "folders", "parent_id")
	if err != nil || ok {
		t.Fatalf("parent_id after v1: ok=%v err=%v", ok, err)
	}

	var folderPath, repoPath string
	if err := db2.Raw(`SELECT path FROM folders WHERE id = 1`).Scan(&folderPath).Error; err != nil || folderPath != "work" {
		t.Fatalf("folder path flattened: got %q err=%v", folderPath, err)
	}
	if err := db2.Raw(`SELECT repo_path FROM repositories WHERE id = 1`).Scan(&repoPath).Error; err != nil || repoPath != "work/api.git" {
		t.Fatalf("repo_path: got %q err=%v", repoPath, err)
	}

	ok, err = migrate.ColumnExists(db2, "repositories", "mirror_direction")
	if err != nil || !ok {
		t.Fatalf("mirror_direction after v1: ok=%v err=%v", ok, err)
	}
	ok, err = migrate.ColumnExists(db2, "access_tokens", "expires_at")
	if err != nil || !ok {
		t.Fatalf("expires_at after v1: ok=%v err=%v", ok, err)
	}
	ok, err = migrate.ColumnExists(db2, "sessions", "ip")
	if err != nil || !ok {
		t.Fatalf("sessions.ip after migrate: ok=%v err=%v", ok, err)
	}
	ok, err = migrate.ColumnExists(db2, "sessions", "user_agent")
	if err != nil || !ok {
		t.Fatalf("sessions.user_agent after migrate: ok=%v err=%v", ok, err)
	}
	ok, err = migrate.TableExists(db2, "folder_members")
	if err != nil || !ok {
		t.Fatalf("folder_members after v1: ok=%v err=%v", ok, err)
	}

	var issueSeq, pullSeq int
	if err := db2.Raw(`SELECT issue_seq, pull_seq FROM repositories WHERE id = 1`).Row().Scan(&issueSeq, &pullSeq); err != nil {
		t.Fatal(err)
	}
	if issueSeq != 3 || pullSeq != 2 {
		t.Fatalf("seqs issue=%d pull=%d", issueSeq, pullSeq)
	}

	var versions int64
	if err := db2.Model(&migrate.Record{}).Count(&versions).Error; err != nil || int(versions) != len(migrations.All()) {
		t.Fatalf("journal versions=%d want %d err=%v", versions, len(migrations.All()), err)
	}
}

func TestBackupSQLite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := filepath.Join(dir, "db.sqlite")
	if err := os.WriteFile(src, []byte("sqlite-demo"), 0o600); err != nil {
		t.Fatal(err)
	}

	dst, err := migrate.BackupSQLite(src)
	if err != nil {
		t.Fatal(err)
	}

	if dst == "" {
		t.Fatal("expected backup path")
	}

	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "sqlite-demo" {
		t.Fatalf("backup body=%q", body)
	}
}

func TestSoftDeleteAllowsNameReuse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	users := NewUserRepo(db)
	u := &domain.User{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "x",
	}
	if err := users.Create(u); err != nil {
		t.Fatal(err)
	}

	first := &domain.Repository{
		Name:          "demo",
		OwnerKind:     domain.OwnerKindUser,
		OwnerID:       u.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "demo.git"),
		RepoPath:      "demo.git",
	}
	repos := NewRepositoryRepo(db)
	if err := repos.Create(first); err != nil {
		t.Fatal(err)
	}
	if err := repos.Delete(first.ID); err != nil {
		t.Fatal(err)
	}

	second := &domain.Repository{
		Name:          "demo",
		OwnerKind:     domain.OwnerKindUser,
		OwnerID:       u.ID,
		DefaultBranch: "main",
		Path:          filepath.Join(dir, "demo2.git"),
		RepoPath:      "demo.git",
	}
	if err := repos.Create(second); err != nil {
		t.Fatalf("reuse after soft-delete: %v", err)
	}
}

func openRawDB(dbPath string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, err
	}

	dsn := dbPath
	if !strings.Contains(dsn, "_pragma") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)"
	}
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	return db, nil
}

func tableColumns(t *testing.T, db *gorm.DB, table string) map[string]bool {
	t.Helper()

	var rows []struct {
		Name string
	}
	if err := db.Raw(fmt.Sprintf(`SELECT name FROM pragma_table_info('%s')`, table)).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}

	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Name] = true
	}

	return out
}

var v2RepoColumns = []string{"mirror_enabled", "mirror_interval", "mirror_sync_window", "mirror_force", "mirror_prune", "mirror_refspecs", "mirror_ssh_private_key", "mirror_ssh_passphrase", "mirror_insecure_tls", "mirror_pinned_hosts", "mirror_lfs", "mirror_max_bytes", "mirror_conflict_policy", "mirror_status", "mirror_fail_count", "mirror_next_sync_at", "mirror_last_progress", "mirror_remote_head", "mirror_owner_only", "mirror_group_id"}

func TestMigrateV2UpgradesV1State(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	fresh, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	wantCols := tableColumns(t, fresh, "repositories")
	for _, c := range v2RepoColumns {
		if !wantCols[c] {
			t.Fatalf("bootstrap schema lacks %s", c)
		}
	}

	steps := []string{
		`DROP INDEX IF EXISTS idx_repositories_mirror`,
		`DROP INDEX IF EXISTS idx_repositories_mirror_due`,
	}
	for _, c := range v2RepoColumns {
		steps = append(steps, "ALTER TABLE repositories DROP COLUMN "+c)
	}

	for _, c := range []string{"mirror_partial", "mirror_verify_signed"} {
		steps = append(steps, "ALTER TABLE repositories DROP COLUMN "+c)
	}

	steps = append(steps,
		`DROP TABLE mirror_sync_events`, `DROP TABLE mirror_targets`, `DROP TABLE mirror_groups`,
		`DELETE FROM schema_migrations`,
		`INSERT INTO repositories (name, owner_id, path, repo_path, mirror_direction, mirror_url, mirror_last_error)
		 VALUES ('legacy', 1, '/tmp/legacy.git', 'legacy.git', 'pull', 'https://example.com/a.git', 'boom')`,
		`INSERT INTO repositories (name, owner_id, path, repo_path, mirror_direction, mirror_url, mirror_last_sync_at)
		 VALUES ('healthy', 1, '/tmp/healthy.git', 'healthy.git', 'pull', 'https://example.com/b.git', '2026-01-01 00:00:00')`,
		`INSERT INTO repositories (name, owner_id, path, repo_path) VALUES ('plain', 1, '/tmp/plain.git', 'plain.git')`,
	)

	for _, stmt := range steps {
		if err := fresh.Exec(stmt).Error; err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}

	if ok, _ := migrate.ColumnExists(fresh, "repositories", "mirror_status"); ok {
		t.Fatal("rewind failed")
	}

	sqlDB, _ := fresh.DB()
	_ = sqlDB.Close()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatalf("migrate v2: %v", err)
	}

	sqlDB, _ = db.DB()
	defer sqlDB.Close()

	got := tableColumns(t, db, "repositories")
	for c := range wantCols {
		if !got[c] {
			t.Fatalf("migrated schema lacks %s", c)
		}
	}

	for _, tbl := range []string{"mirror_sync_events", "mirror_targets", "mirror_groups"} {
		if ok, err := migrate.TableExists(db, tbl); err != nil || !ok {
			t.Fatalf("%s missing: ok=%v err=%v", tbl, ok, err)
		}
	}

	type row struct {
		RepoPath      string
		MirrorStatus  string
		MirrorEnabled bool
		MirrorForce   bool
		FailCount     int
	}

	var rows []row
	if err := db.Raw(`SELECT repo_path, mirror_status, mirror_enabled, mirror_force, mirror_fail_count AS fail_count FROM repositories ORDER BY id`).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"legacy.git": "error", "healthy.git": "ok", "plain.git": ""}
	for _, r := range rows {
		if r.MirrorStatus != want[r.RepoPath] {
			t.Fatalf("%s: status=%q want %q", r.RepoPath, r.MirrorStatus, want[r.RepoPath])
		}

		if !r.MirrorEnabled || !r.MirrorForce || r.FailCount != 0 {
			t.Fatalf("%s: defaults not applied: %+v", r.RepoPath, r)
		}
	}

	var n int64
	if err := db.Model(&migrate.Record{}).Count(&n).Error; err != nil || int(n) != len(migrations.All()) {
		t.Fatalf("journal=%d want %d err=%v", n, len(migrations.All()), err)
	}

	ctx := context.Background()
	if c, err := NewRepositoryRepo(db).CountMirrorsInError(ctx, time.Now()); err != nil || c != 1 {
		t.Fatalf("CountMirrorsInError=%d err=%v", c, err)
	}

	if err := NewMirrorSyncRepo(db).Create(ctx, &domain.MirrorSyncEvent{RepoID: 1, Direction: "pull", Status: domain.MirrorEventOK, StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	_ = sqlDB.Close()

	again, err := NewDB(dir)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}

	if err := VerifySchema(again); err != nil {
		t.Fatalf("schema after v2 upgrade: %v", err)
	}

	sqlDB, _ = again.DB()
	_ = sqlDB.Close()
}

func TestSchemaMatchesModelsFresh(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err := VerifySchema(db); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaDetectsMissingColumn(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Exec(`ALTER TABLE repositories DROP COLUMN mirror_partial`).Error; err != nil {
		t.Fatal(err)
	}

	err = VerifySchema(db)
	if err == nil || !strings.Contains(err.Error(), "mirror_partial") {
		t.Fatalf("want missing mirror_partial, got %v", err)
	}

	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func TestMigrateV3UpgradesV2State(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}

	v3Cols := []string{"mirror_partial", "mirror_verify_signed"}
	for _, c := range v3Cols {
		if err := db.Exec("ALTER TABLE repositories DROP COLUMN " + c).Error; err != nil {
			t.Fatalf("drop %s: %v", c, err)
		}
	}
	if err := db.Exec(`DELETE FROM schema_migrations`).Error; err != nil {
		t.Fatal(err)
	}

	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	db2, err := NewDB(dir)
	if err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	sqlDB2, err := db2.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB2.Close()

	for _, c := range v3Cols {
		ok, err := migrate.ColumnExists(db2, "repositories", c)
		if err != nil || !ok {
			t.Fatalf("%s after v3: ok=%v err=%v", c, ok, err)
		}
	}

	if err := VerifySchema(db2); err != nil {
		t.Fatal(err)
	}

	var versions int64
	if err := db2.Model(&migrate.Record{}).Count(&versions).Error; err != nil || int(versions) != len(migrations.All()) {
		t.Fatalf("journal=%d want %d err=%v", versions, len(migrations.All()), err)
	}
}
