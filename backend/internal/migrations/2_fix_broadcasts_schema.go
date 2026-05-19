package migrations

import (
	"log"

	"github.com/pocketbase/dbx"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migration 2: Drop the raw broadcasts table if it exists.
// Previously, this migration created a raw SQLite table, but that bypasses
// PocketBase's collection registry (_collections), leading to 404 errors in the API.
// We now drop the raw table here and let init.go recreate it properly via JSON schema import.
func init() {
	m.Register(func(db dbx.Builder) error {
		log.Println("🔄 [Migration 2] Dropping raw broadcasts table to allow proper collection recreation...")

		steps := []string{
			"DROP TABLE IF EXISTS broadcasts",
		}

		for _, sql := range steps {
			if _, err := db.NewQuery(sql).Execute(); err != nil {
				log.Printf("❌ [Migration 2] Failed SQL step: %v", err)
				return err
			}
		}

		log.Println("✅ [Migration 2] Raw broadcasts table dropped successfully!")
		return nil
	}, func(db dbx.Builder) error {
		return nil
	})
}

