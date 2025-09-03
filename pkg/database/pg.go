package database

import (
	"fmt"
	"log"
	"sync"

	"github.com/crm/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db          *gorm.DB
	err         error
	client_once sync.Once
)

func InitDB(dbc config.Database) {
	client_once.Do(func() {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbc.Host, dbc.Port, dbc.User, dbc.Pass, dbc.Name)
		db, err = gorm.Open(
			postgres.New(
				postgres.Config{
					DSN:                  dsn,
					PreferSimpleProtocol: true, // daha az kaynak
				},
			),
			&gorm.Config{
				DisableForeignKeyConstraintWhenMigrating: false,
			},
		)
		if err != nil {
			log.Printf("[error] failed to initialize database, got error %v", err)
			panic(err)
		}

		// Get the underlying SQL database connection to execute raw SQL
		sqlDB, err := db.DB()
		if err != nil {
			log.Printf("[error] failed to get underlying database connection: %v", err)
			panic(err)
		}

		// Test the connection
		if err := sqlDB.Ping(); err != nil {
			log.Printf("[error] failed to ping database: %v", err)
			panic(err)
		}

		log.Printf("[info] database connection established successfully")

		// Run migrations
		if err := AutoMigrate(db); err != nil {
			log.Printf("Migration failed: %v", err)
			panic(err)
		}

		log.Printf("[info] database migrations completed successfully")
	})
}

func DBClient() *gorm.DB {
	if db == nil {
		log.Panic("Postgres is not initialized. Call InitDB first.")
	}
	return db
}
