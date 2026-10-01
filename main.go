package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"vaultorb/internal/db"

	tea "charm.land/bubbletea/v2"

	_ "modernc.org/sqlite"
)

//go:embed wordlist.txt
var largeWordlist string

//go:embed internal/db/schema.sql
var ddl string

type state struct {
	masterKey []byte
	dbPath    string
	db        *db.Queries
	wordlist  []string
}

// dbPath should be deleted if there's no use for it but keep for now
type services struct {
	dbQ        *db.Queries
	wordlist   []string
	dbPath     string
	isFirstRun bool
}

func main() {
	fmt.Println("VaultOrb CLI initialized!")

	fmt.Println("connecting to database...")
	dbPath := "./vault.db"
	_, err := os.Stat(dbPath)
	isFirstRun := os.IsNotExist(err)

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}

	defer conn.Close()

	if err := conn.Ping(); err != nil {
		log.Fatalf("error pinging database: %v", err)
	}
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		log.Fatalf("failed to init database schema: %v", err)
	}
	dbQueries := db.New(conn)

	wordlist := strings.Split(largeWordlist, "\n")
	//dbpath also here
	services := services{
		dbQ:        dbQueries,
		wordlist:   wordlist,
		dbPath:     dbPath,
		isFirstRun: isFirstRun,
	}

	p := tea.NewProgram(initialModel(services))

	_, err = p.Run()
	if err != nil {
		log.Fatalf("there has been a problem")
	}

}
