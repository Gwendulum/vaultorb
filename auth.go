package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"vaultorb/internal/db"

	"golang.org/x/crypto/argon2"
)

func unlockVault(ctx context.Context, dbQ *db.Queries, password []byte) ([]byte, error) {

	salt, err := getOrCreateSalt(ctx, dbQ)
	if err != nil {
		return nil, err
	}

	masterKey := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
	//fetch encrypted phrase from database. If it exists, error is nil and the indented block is skipped.
	encryptedPhrase, err := dbQ.GetMetadata(ctx, "vault_check")
	//if we get an error that means either there's no password or the database failed in some other way
	if err != nil {
		//Guard clause. If this fails the error is something else than a missing encrypted phrase and we return early.
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("Failed to retrieve phrase: %w", err)
		}
		// create new encrypted phrase using master key and and our "vault_ok" dummy phrase
		cipherText, err := encrypt(masterKey, []byte("vault_ok"))
		if err != nil {
			return nil, fmt.Errorf("Encryption: %w", err)
		}
		//store new encrypted phrase for next time
		err = dbQ.SetMetadata(ctx, db.SetMetadataParams{Key: "vault_check", Value: cipherText})
		if err != nil {
			return nil, fmt.Errorf("Failed to store Canary in database: %w", err)
		}
		//return masterKey, don't need to validate
		return masterKey, nil
	}
	//decrypt our stored phrase using our generated masterKey. gcm.Open() also validates our masterKey against the encrypted phrase, removing the need for further checks
	_, err = decrypt(masterKey, encryptedPhrase)
	if err != nil {
		return nil, fmt.Errorf("Decyption: %w", err)
	}

	return masterKey, nil
}

func getOrCreateSalt(ctx context.Context, dbQ *db.Queries) ([]byte, error) {
	salt, err := dbQ.GetMetadata(ctx, "argon2_salt")
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to get salt from db: %w", err)
		}
		newSalt := make([]byte, 16)
		if _, err := rand.Read(newSalt); err != nil {
			return nil, fmt.Errorf("failed to generate random salt: %w", err)
		}
		err = dbQ.SetMetadata(ctx, db.SetMetadataParams{Key: "argon2_salt", Value: newSalt})
		if err != nil {
			return nil, fmt.Errorf("failed to set new salt: %w", err)
		}
		return newSalt, nil
	}

	return salt, nil

}

func encrypt(masterKey []byte, phrase []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("Failed creating cipherBlock: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("Failed creating gcm: %w", err)
	}
	// Create the "Random number used Once" and allocate the required memory for passing into gcm.Seal's 'dst'(destination) parameter, preventing unnecessary reallocation.
	randNonce := make([]byte, gcm.NonceSize(), gcm.NonceSize()+len(phrase)+gcm.Overhead())
	if _, err := rand.Read(randNonce); err != nil {
		return nil, fmt.Errorf("failed to generate random nonce: %w", err)
	}
	cipherText := gcm.Seal(randNonce, randNonce, phrase, nil)

	return cipherText, nil

}

func decrypt(masterKey []byte, phrase []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("Failed creating cipherBlock: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("Failed creating gcm: %w\n", err)
	}

	if len(phrase) < gcm.NonceSize() {
		return nil, fmt.Errorf("Malformed cipher. It's too short")
	}

	nonce := phrase[:gcm.NonceSize()]
	cipherText := phrase[gcm.NonceSize():]

	decryptedPhrase, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed decrypting cipher: %w", err)
	}

	return decryptedPhrase, nil
}
