package main

import (
	tea "charm.land/bubbletea/v2"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"strings"
	"vaultorb/internal/db"

	_ "modernc.org/sqlite"
)

//go:embed wordlist.txt
var largeWordlist string

type state struct {
	masterKey []byte
	dbPath    string
	db        *db.Queries
	wordlist  []string
}

// dbPath should be deleted if there's no use for it but keep for now
type services struct {
	dbQ      *db.Queries
	wordlist []string
	dbPath   string
}

func main() {
	fmt.Println("VaultOrb CLI initialized!")

	fmt.Println("connecting to database...")
	dbPath := "./vault.db"
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}

	defer conn.Close()
	if err := conn.Ping(); err != nil {
		log.Fatalf("error pinging database: %v", err)
	}

	dbQueries := db.New(conn)

	wordlist := strings.Split(largeWordlist, "\n")
	//dbpath also here
	services := services{
		dbQ:      dbQueries,
		wordlist: wordlist,
		dbPath:   dbPath,
	}

	p := tea.NewProgram(initialModel(services))

	_, err = p.Run()
	if err != nil {
		log.Fatalf("there has been a problem")
	}

}
