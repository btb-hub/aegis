package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("db pool: %w", err)
	}
	defer pool.Close()

	store := db.NewStore(pool)
	doc, err := store.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("initialize settings by starting the local API before seeding users")
	}
	if err := config.SeedDevAllowed(doc.Values["PUBLIC_URL"]); err != nil {
		return err
	}
	users, err := store.SeedDevUsers(ctx)
	if err != nil {
		return fmt.Errorf("seed dev users: %w", err)
	}

	for _, user := range users {
		log.Printf("seeded user %s (%s) role=%s provider=%s", user.DisplayName, user.Email, user.Role, user.Provider)
	}
	log.Printf("seeded %d dev users", len(users))
	return nil
}
