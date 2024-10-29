package models

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	ent "github.com/doncicuto/openuem_ent"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

type Model struct {
	Client *ent.Client
}

func New(databaseType, dbUrl string) (*Model, error) {
	model := Model{}

	if databaseType == "SQLite" {
		if _, err := os.Stat(dbUrl); err != nil {
			_, err := os.Create(dbUrl)
			if err != nil {
				return nil, err
			}
		}

		db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&_fk=1", dbUrl))
		if err != nil {
			return nil, fmt.Errorf("could not connect with SQLite database: %v", err)
		}

		model.Client = ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	} else {
		db, err := sql.Open("pgx", dbUrl)
		if err != nil {
			return nil, fmt.Errorf("could not connect with Postgres database: %v", err)
		}

		model.Client = ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	}

	// TODO Automatic migrations only in development
	ctx := context.Background()
	if os.Getenv("ENV") != "prod" {
		if err := model.Client.Schema.Create(ctx); err != nil {
			return nil, err
		}
	}

	return &model, nil
}

func (m *Model) Close() {
	m.Client.Close()
}
