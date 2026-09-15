package arch

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV15CoreDoesNotImportPostgresDriver(t *testing.T) {
	roots := []string{
		"../../internal/controlplane",
		"../../internal/engineering",
		"../../internal/taskexecution",
		"../../internal/persistence",
	}
	forbidden := []string{
		`"github.com/jackc/pgx/`,
		`"github.com/jackc/pgx/v5"`,
		`"github.com/jackc/pgx/v5/pgtype"`,
	}

	for _, root := range roots {
		root := root
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				// Concrete adapters own database driver imports. The Core boundary
				// above them must remain database-neutral.
				if path != root && filepath.Dir(path) == filepath.Clean("../../internal/persistence") {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, needle := range forbidden {
				if strings.Contains(string(data), needle) {
					t.Errorf("v1.5 Core file %s leaks PostgreSQL driver import %s", path, needle)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
