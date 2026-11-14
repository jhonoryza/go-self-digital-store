// Package db membuka SQLite (modernc.org/sqlite, pure-Go), menjalankan
// migrasi embedded, dan backup berkala (VACUUM INTO).
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open buka database di <dir>/store.db dengan pragma WAL + busy timeout.
func Open(dir string) (*sql.DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	d, err := sql.Open("sqlite", filepath.Join(dir, "store.db")+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	// SQLite menulis eksklusif; satu koneksi menghindari SQLITE_BUSY senyap.
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)
	return d, d.Ping()
}

// Migrate jalankan seluruh migrasi NNNN_*.sql yang belum diterapkan.
func Migrate(d *sql.DB) error {
	if _, err := d.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))`); err != nil {
		return err
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var version int
		if _, err := fmt.Sscanf(name, "%04d", &version); err != nil {
			return fmt.Errorf("nama migrasi %q tak sesuai NNNN_", name)
		}
		var applied int
		if err := d.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}
		b, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(b)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrasi %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("migrasi %s diterapkan", name)
	}
	return nil
}

// Backup salin database online (VACUUM INTO) ke <dir>/backups/.
func Backup(d *sql.DB, dir string, keep int) error {
	bd := filepath.Join(dir, "backups")
	if err := os.MkdirAll(bd, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(bd, "store-"+time.Now().UTC().Format("20060102-150405")+".db")
	if _, err := d.Exec(`VACUUM INTO ?`, dst); err != nil {
		return err
	}
	old, _ := filepath.Glob(filepath.Join(bd, "store-*.db"))
	sort.Strings(old)
	for i := 0; i < len(old)-keep; i++ {
		os.Remove(old[i])
	}
	return nil
}
