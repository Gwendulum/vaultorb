package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
	"unicode"
	"vaultorb/internal/db"
)

func registerPassword(ctx context.Context, dbQ *db.Queries, args []string, masterKey []byte) (string, error) {
	if len(args) != 3 {
		return "", fmt.Errorf("new entry must include domain, username, and password\n")

	}
	domain := args[0]
	username := args[1]
	password := args[2]

	encryptedPassword, err := encrypt(masterKey, []byte(password))
	if err != nil {
		return "", fmt.Errorf("Register func: %w", err)
	}
	entry, err := dbQ.CreateEntry(ctx, db.CreateEntryParams{
		Domain:            domain,
		Username:          username,
		EncryptedPassword: encryptedPassword,
		CreatedAt:         time.Now().UTC(),
	})
	if err != nil {
		return "", fmt.Errorf("error creating new entry: %v\n", err)
	}
	return fmt.Sprintf("New password for %v: %v entry created successfully!\n", entry.Domain, entry.Username), nil

}

func getPassword(ctx context.Context, dbQ *db.Queries, args []string, masterKey []byte) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("Password retrieval requires domain and username\n")
	}
	domain := args[0]
	username := args[1]
	password, err := dbQ.GetEntry(ctx, db.GetEntryParams{
		Domain:   domain,
		Username: username,
	})
	if err != nil {
		return "", fmt.Errorf("error retrieving password\n")
	}
	decryptedPassword, err := decrypt(masterKey, password)
	return string(decryptedPassword), nil

}

func listPassword(ctx context.Context, dbQ *db.Queries) ([]db.Entry, error) {
	list, err := dbQ.ListEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("cmdList: %w", err)
	}

	return list, nil
}

func deletePassword(ctx context.Context, dbQ *db.Queries, args []string) (db.Entry, error) {
	if len(args) != 2 {
		return db.Entry{}, fmt.Errorf("Entry deletion requires domain and username\n")
	}

	domain := args[0]
	username := args[1]

	entry, err := dbQ.DeleteEntry(ctx, db.DeleteEntryParams{
		Domain:   domain,
		Username: username,
	})
	if err != nil {
		return db.Entry{}, fmt.Errorf("error deleting entry\n")
	}
	return entry, nil
}

func generatePassword(svc services) (string, error) {
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

	maxWords := big.NewInt(int64(len(svc.wordlist)))
	maxSymbols := big.NewInt(int64(len(symbolsList)))
	maxInt := big.NewInt(int64(len(intList)))
	coinflipMax := big.NewInt(int64(2))

	for i := 0; i < 4; i++ {
		randWord, err = rand.Int(rand.Reader, maxWords)
		if err != nil {
			return "", fmt.Errorf("Failed rand.Int generation: %w", err)
		}
		password = append(password, svc.wordlist[randWord.Int64()])

		coinflip, err = rand.Int(rand.Reader, coinflipMax)
		if err != nil {
			return "", fmt.Errorf("Failed coinflip rand.Int generation: %w", err)
		}
		if coinflip.Int64() == 0 {
			randSymbol, err = rand.Int(rand.Reader, maxSymbols)
			if err != nil {
				return "", fmt.Errorf("Failed symbol rand.Int generation: %w", err)
			}
			password = append(password, symbolsList[randSymbol.Int64()])
			hasSymbol = true

		} else if coinflip.Int64() == 1 {

			randInt, err = rand.Int(rand.Reader, maxInt)
			if err != nil {
				return "", fmt.Errorf("Failed integer rand.Int generation: %w", err)
			}
			password = append(password, intList[randInt.Int64()])
			hasInteger = true
		}
	}

	if !hasSymbol {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(password)/2)))
		if err != nil {
			return "", fmt.Errorf("Failed index rand.Int generation: %w", err)
		}
		randSymbol, err = rand.Int(rand.Reader, maxSymbols)
		if err != nil {
			return "", fmt.Errorf("Failed integer rand.Int generation: %w", err)
		}
		password[(randIndex.Int64()*2)+1] = symbolsList[randSymbol.Int64()]
		hasSymbol = true
	}

	if !hasInteger {
		randIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(password)/2)))
		if err != nil {
			return "", fmt.Errorf("Failed index rand.Int generation: %w", err)
		}
		randInt, err = rand.Int(rand.Reader, maxInt)
		if err != nil {
			return "", fmt.Errorf("Failed integer rand.Int generation: %w", err)
		}
		password[(randIndex.Int64()*2)+1] = intList[randInt.Int64()]

		hasInteger = true
	}

	var passWithSuffix string
	for i := 0; i < 8; i += 2 {
		passWithSuffix = passWithSuffix + CapitalizeFirstLetter(password[i]) + password[i+1]
		if i != 6 {
			passWithSuffix = passWithSuffix + "-"
		}
	}

	return passWithSuffix, nil
}

func CapitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return ""
	}
	runes := []rune(s)

	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}
