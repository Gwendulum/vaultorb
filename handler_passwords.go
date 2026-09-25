package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
	"unicode"
	"vaultorb/internal/db"
)

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Args) != 3 {
		return fmt.Errorf("new entry must include domain, username, and password\n")

	}
	domain := cmd.Args[0]
	username := cmd.Args[1]
	password := cmd.Args[2]

	encryptedPassword, err := encrypt(s.masterKey, []byte(password))
	if err != nil {
		return fmt.Errorf("Register func: %w", err)
	}
	entry, err := s.db.CreateEntry(context.Background(), db.CreateEntryParams{
		Domain:            domain,
		Username:          username,
		EncryptedPassword: encryptedPassword,
		CreatedAt:         time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("error creating new entry: %v\n", err)
	}
	fmt.Printf("New password for %v entry created successfully!\n", entry.Domain)
	return nil

}

func handlerGet(s *state, cmd command) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("Password retrieval requires domain and username\n")

	}

	domain := cmd.Args[0]
	username := cmd.Args[1]

	password, err := s.db.GetEntry(context.Background(), db.GetEntryParams{
		Domain:   domain,
		Username: username,
	})
	if err != nil {
		return fmt.Errorf("error retrieving password\n")
	}
	encryptedPassword := base64.StdEncoding.EncodeToString(password)
	fmt.Printf("encrypted: %s\n", string(encryptedPassword))
	decryptedPassword, err := decrypt(s.masterKey, password)
	fmt.Printf("password for %s: %s is: %v\n", domain, username, string(decryptedPassword))
	return nil

}

func handlerList(s *state, cmd command) error {
	fmt.Printf("listing...\n")
	list, err := s.db.ListEntries(context.Background())
	if err != nil {
		return fmt.Errorf("cmdList: %w", err)
	}
	for _, dbEntry := range list {
		fmt.Printf("Domain: %s, Username: %s, Created at: %v\n", dbEntry.Domain, dbEntry.Username, dbEntry.CreatedAt)
	}

	return nil
}

func handlerDelete(s *state, cmd command) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("Entry deletion requires domain and username\n")
	}

	domain := cmd.Args[0]
	username := cmd.Args[1]

	entry, err := s.db.DeleteEntry(context.Background(), db.DeleteEntryParams{
		Domain:   domain,
		Username: username,
	})
	if err != nil {
		return fmt.Errorf("error deleting entry\n")
	}
	fmt.Printf("Entry for %s: %s has been deleted\n", entry.Domain, entry.Username)
	return nil
}

func handlerClear(s *state, cmd command) error {
	err := s.db.ClearEntries(context.Background())
	if err != nil {
		return fmt.Errorf("errror clearing entries: %w", err)
	}
	fmt.Printf("all entries deleted\n")
	return nil
}

func handlerGenerate(s *state, cmd command) error {
	var randWord *big.Int
	var randSymbol *big.Int
	var randInt *big.Int
	var coinflip *big.Int
	var err error
	var password []string
	var hasSymbol bool
	var hasInteger bool

	symbolsList := []string{"!", "#", "%", "&", "/", "+", "_", "£", "$", "€"}

	intList := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}

	maxWords := big.NewInt(int64(len(s.wordlist)))
	maxSymbols := big.NewInt(int64(len(symbolsList)))
	maxInt := big.NewInt(int64(len(intList)))
	coinflipMax := big.NewInt(int64(2))

	for i := 0; i < 4; i++ {
		randWord, err = rand.Int(rand.Reader, maxWords)
		if err != nil {
			return fmt.Errorf("Failed rand.Int generation: %w", err)
		}
		password = append(password, s.wordlist[randWord.Int64()])

		coinflip, err = rand.Int(rand.Reader, coinflipMax)
		if err != nil {
			return fmt.Errorf("Failed coinflip rand.Int generation: %w", err)
		}
		if coinflip.Int64() == 0 {
			randSymbol, err = rand.Int(rand.Reader, maxSymbols)
			if err != nil {
				return fmt.Errorf("Failed symbol rand.Int generation: %w", err)
			}
			password = append(password, symbolsList[randSymbol.Int64()])
			hasSymbol = true

		} else if coinflip.Int64() == 1 {

			randInt, err = rand.Int(rand.Reader, maxInt)
			if err != nil {
				return fmt.Errorf("Failed integer rand.Int generation: %w", err)
			}
			password = append(password, intList[randInt.Int64()])
			hasInteger = true
		}
	}

	if !hasSymbol {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(password)/2)))
		if err != nil {
			return fmt.Errorf("Failed index rand.Int generation: %w", err)
		}
		randSymbol, err = rand.Int(rand.Reader, maxSymbols)
		if err != nil {
			return fmt.Errorf("Failed integer rand.Int generation: %w", err)
		}
		password[(randIndex.Int64()*2)+1] = symbolsList[randSymbol.Int64()]
		hasSymbol = true
	}

	if !hasInteger {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(password)/2)))
		if err != nil {
			return fmt.Errorf("Failed index rand.Int generation: %w", err)
		}
		randInt, err = rand.Int(rand.Reader, maxInt)
		if err != nil {
			return fmt.Errorf("Failed integer rand.Int generation: %w", err)
		}
		password[(randIndex.Int64()*2)+1] = intList[randInt.Int64()]

		hasInteger = true
	}

	var passWithSuffix string
	for i := 0; i < 8; i += 2 {
		passWithSuffix = passWithSuffix + CapitalizeFirst(password[i]) + password[i+1]
		if i != 6 {
			passWithSuffix = passWithSuffix + "-"
		}
	}

	fmt.Printf("your new password is: %s\n", passWithSuffix)
	return nil
}

func CapitalizeFirst(s string) string {
	if len(s) == 0 {
		return ""
	}
	runes := []rune(s)

	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}
