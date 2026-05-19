package migrations

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

//go:embed pb_schema.json
var schemaBytes []byte

func init() {
	m.Register(func(db dbx.Builder) error {
		log.Println("🔄 [Migration] Starting automatic schema import from pb_schema.json...")
		
		// 1. Instantiate the Dao from the transaction builder
		dao := daos.New(db)
		
		// 2. Unmarshal the schema JSON into a slice of Collection models
		var collections []*models.Collection
		if err := json.Unmarshal(schemaBytes, &collections); err != nil {
			log.Printf("❌ [Migration] Failed to parse pb_schema.json: %v", err)
			return err
		}
		
		// 3. Query existing default users collection to preserve it
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
				log.Println("ℹ️ [Migration] Appended default system 'users' collection to maintain schema integrity")
			}
		} else {
			log.Printf("ℹ️ [Migration] Default 'users' collection was not found in database or error: %v", err)
		}
		
		// 4. Programmatically add 'user_id' relation field pointing to users collection in students and managers
		for _, col := range collections {
			if col.Name == "students" || col.Name == "managers" {
				hasUserId := false
				for _, f := range col.Schema.Fields() {
					if f.Name == "user_id" {
						hasUserId = true
						break
					}
				}
				
				if !hasUserId {
					maxSelect := 1
					userIdField := &schema.SchemaField{
						Id:       "fld_" + col.Name + "_user",
						Name:     "user_id",
						Type:     schema.FieldTypeRelation,
						Required: false,
						Options: &schema.RelationOptions{
							MaxSelect:     &maxSelect,
							CollectionId:  "_pb_users_auth_",
							CascadeDelete: true,
						},
					}
					col.Schema.AddField(userIdField)
					log.Printf("✅ [Migration] Added programmatic 'user_id' relation field to '%s' collection", col.Name)
				}
			}
		}
		
		// 5. Import all collections without deleting collections not present in the JSON schema.
		// broadcasts is managed by a separate raw SQL migration (2_fix_broadcasts_schema.go).
		err = dao.ImportCollections(collections, false, nil)
		if err != nil {
			log.Printf("❌ [Migration] Failed to import collections: %v", err)
			return err
		}
		log.Println("✅ [Migration] Schema collections imported successfully!")
		
		// 6. Auto-provision the superuser (Admin) from environment variables
		adminEmail := os.Getenv("PB_ADMIN_EMAIL")
		adminPassword := os.Getenv("PB_ADMIN_PASSWORD")
		
		if adminEmail != "" && adminPassword != "" {
			_, err := dao.FindAdminByEmail(adminEmail)
			if err != nil {
				// Admin doesn't exist, create a new one
				newAdmin := &models.Admin{}
				newAdmin.Email = adminEmail
				if err := newAdmin.SetPassword(adminPassword); err != nil {
					log.Printf("❌ [Migration] Failed to set password for default admin: %v", err)
					return err
				}
				
				if err := dao.SaveAdmin(newAdmin); err != nil {
					log.Printf("❌ [Migration] Failed to save default admin: %v", err)
					return err
				}
				log.Printf("👑 [Migration] Default admin (%s) created successfully!", adminEmail)
			} else {
				log.Printf("ℹ️ [Migration] Default admin (%s) already exists.", adminEmail)
			}
		} else {
			log.Println("⚠️ [Migration] PB_ADMIN_EMAIL and PB_ADMIN_PASSWORD are not defined in environmental variables")
		}
		
		return nil
	}, func(db dbx.Builder) error {
		// Down migration (no-op)
		return nil
	})
}
