// config-import reads production dotenv as data and prints key/status pairs only.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("config-import", flag.ContinueOnError)
	// Flag parsing must not echo arbitrary argument values into terminal output.
	flags.SetOutput(io.Discard)
	file := flags.String("env-file", "", "path to the existing production .env file")
	dry := flags.Bool("dry-run", false, "validate and report without writes")
	apply := flags.Bool("apply", false, "apply the validated import atomically")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("invalid importer arguments; use --env-file PATH and --dry-run or --apply")
	}
	if *file == "" || *dry == *apply || flags.NArg() != 0 {
		return fmt.Errorf("use --env-file PATH and exactly one of --dry-run or --apply")
	}
	f, err := os.Open(*file)
	if err != nil {
		return fmt.Errorf("cannot open env file")
	}
	defer f.Close()
	values, err := config.ParseEnv(f)
	if err != nil {
		return err
	}
	connection := os.Getenv("DATABASE_URL")
	if connection == "" {
		connection = values["DATABASE_URL"]
	}
	if connection == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, connection)
	if err != nil {
		return fmt.Errorf("cannot configure database connection")
	}
	defer pool.Close()
	report, err := db.NewStore(pool).ImportEnvironment(ctx, values, *apply)
	for _, item := range report {
		fmt.Fprintf(out, "%s\t%s\n", item.Key, item.Status)
	}
	if err != nil {
		return err
	}
	if *dry {
		fmt.Fprintln(out, "Dry run complete. No changes written.")
	} else {
		fmt.Fprintln(out, "Import complete. Database settings are authoritative.")
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
