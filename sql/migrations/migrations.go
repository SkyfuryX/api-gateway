package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

func RunMigrations(log *log.Logger, dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open DB for migrations: %v", err)
	}
	defer func() {
		if err = db.Close(); err != nil{
			fmt.Println(err)
		}
	}()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %v", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}
	log.Print("Migrations successfully applied")
	return nil
}
