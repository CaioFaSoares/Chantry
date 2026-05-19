package migrations

import (
	"encoding/json"
	"log"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

// Migration 3: Re-import schema to create the broadcasts collection properly.
// Since init.go was already run and is skipped on restarts, and migration 2 dropped
// the raw broadcasts table, we need this migration to re-run the schema import
// from pb_schema.json so PocketBase registers the broadcasts collection internally
// (in _collections) and exposes it via the REST API.
func init() {
	m.Register(func(db dbx.Builder) error {
		log.Println("🔄 [Migration 3] Re-importing schema to register broadcasts collection...")

		dao := daos.New(db)

		var collections []*models.Collection
		if err := json.Unmarshal(schemaBytes, &collections); err != nil {
			log.Printf("❌ [Migration 3] Failed to parse pb_schema.json: %v", err)
			return err
		}

		// Ensure we don't drop extra collections like 'users'
		usersCol, err := dao.FindCollectionByNameOrId("users")
		if err == nil && usersCol != nil {
			found := false
			for _, c := range collections {
				if c.Name == "users" {
					found = true
					break
				}
			}
			if !found {
				collections = append(collections, usersCol)
			}
		}

		// Drop the raw broadcasts table if it exists so ImportCollections can recreate it cleanly
		if _, err := db.NewQuery("DROP TABLE IF EXISTS broadcasts").Execute(); err != nil {
			log.Printf("⚠️ [Migration 3] Failed to drop existing broadcasts table: %v", err)
		}

		// Import the collections (deleteMissing = false)
		if err := dao.ImportCollections(collections, false, nil); err != nil {
			log.Printf("❌ [Migration 3] Failed to import collections: %v", err)
			return err
		}


		log.Println("✅ [Migration 3] Schema re-imported successfully! broadcasts collection is now registered.")
		return nil
	}, func(db dbx.Builder) error {
		return nil
	})
}
