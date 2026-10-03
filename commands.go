package main

import (
	"charm.land/bubbletea/v2"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"vaultorb/internal/db"
)

type checkInitialRunMsg struct {
	isInitialized bool
	err           error
}

func checkInitialRunCmd(dbQ *db.Queries) tea.Cmd {
	return func() tea.Msg {
		var isInit bool
		ctx := context.Background()
		_, err := dbQ.GetMetadata(ctx, "vault_check")
		if err != nil {
			//Guard clause. If this fails the error is something else than a missing encrypted phrase and we return early.
			if !errors.Is(err, sql.ErrNoRows) {
				return checkInitialRunMsg{err: err}
			}
			isInit = false
		} else {
			isInit = true
		}

		return checkInitialRunMsg{isInitialized: isInit}
	}
}

type cmdMsg[T any] struct {
	data T
	err  error
}

type authMsg cmdMsg[[]byte]

func authenticateCmd(dbQ *db.Queries, password string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		masterKey, err := unlockVault(ctx, dbQ, []byte(password))
		if err != nil {
			return authMsg{data: nil, err: fmt.Errorf("authentication failed: %w", err)}
		}
		return authMsg{data: masterKey, err: nil}
	}
}

type getMsg cmdMsg[string]

func getCmd(dbQ *db.Queries, domUser string, masterKey []byte) tea.Cmd {
	return func() tea.Msg {
		splitString := strings.Fields(domUser)
		ctx := context.Background()
		decryptedPassword, err := getPassword(ctx, dbQ, splitString, masterKey)
		if err != nil {
			return getMsg{err: err}
		}
		return getMsg{data: decryptedPassword}
	}
}

type regMsg cmdMsg[db.Entry]

type entryWithPassword struct {
	entry    db.Entry
	password string
	err      error
}

type regUpdateMsg cmdMsg[entryWithPassword]

func registerCmd(dbQ *db.Queries, domUserPass string, masterKey []byte) tea.Cmd {
	return func() tea.Msg {
		splitString := strings.Fields(domUserPass)
		ctx := context.Background()
		entry, err := registerPassword(ctx, dbQ, splitString, masterKey)
		if err != nil {
			if err != sql.ErrNoRows {
				return regMsg{err: err}
			}

			entry, err := getEntry(ctx, dbQ, splitString[:2], masterKey)
			return regUpdateMsg{
				data: entryWithPassword{
					entry:    entry,
					password: splitString[2],
					err:      err,
				},
			}

		}
		return regMsg{data: entry}
	}
}

type listMsg cmdMsg[[]db.Entry]

func listCmd(dbQ *db.Queries) tea.Cmd {
	return func() tea.Msg {
		list, err := listPassword(context.Background(), dbQ)
		if err != nil {
			return listMsg{err: err}
		}
		return listMsg{data: list}
	}
}

type deleteMsg cmdMsg[db.Entry]

func deleteCmd(dbQ *db.Queries, domUser string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		splitString := strings.Fields(domUser)
		entry, err := deletePassword(ctx, dbQ, splitString)
		if err != nil {
			return deleteMsg{err: err}
		}

		return deleteMsg{data: entry}

	}
}

type updateMsg cmdMsg[db.Entry]

func updateCmd(dbQ *db.Queries, domUser string, newPassword string, masterKey []byte) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		splitString := strings.Fields(domUser)
		oldEntry, err := getEntry(ctx, dbQ, splitString, masterKey)
		if err != nil {
			return updateMsg{err: err}
		}
		_, err = updatePassword(ctx, dbQ, newPassword, splitString, masterKey)
		if err != nil {
			return updateMsg{err: err}
		}
		return updateMsg{data: oldEntry}
	}
}

type generateMsg cmdMsg[string]

func generateCmd(svc services) tea.Cmd {
	return func() tea.Msg {
		password, err := generatePassword(svc)
		if err != nil {
			return generateMsg{err: err}
		}
		return generateMsg{data: password}
	}
}

type restoreMsg cmdMsg[historyEntry]

func restoreCmd(dbQ *db.Queries, entry historyEntry) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		restoredRow, err := dbQ.RestoreEntry(ctx, db.RestoreEntryParams{
			ID:                entry.payload.ID,
			Domain:            entry.payload.Domain,
			Username:          entry.payload.Username,
			EncryptedPassword: entry.payload.EncryptedPassword,
			CreatedAt:         entry.payload.CreatedAt,
		})
		if err != nil {
			return restoreMsg{err: err}
		}
		entry.payload = restoredRow
		return restoreMsg{data: entry}
	}
}

type redeleteMsg cmdMsg[historyEntry]

func redeleteCmd(dbQ *db.Queries, entry historyEntry) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		deletedEntry, err := dbQ.DeleteEntry(ctx, db.DeleteEntryParams{
			Domain:   entry.payload.Domain,
			Username: entry.payload.Username,
		})
		if err != nil {
			return redeleteMsg{err: err}
		}
		entry.payload = deletedEntry
		return redeleteMsg{data: entry}
	}
}

type restoreUpdateMsg cmdMsg[historyEntry]

func restoreUpdateCmd(dbQ *db.Queries, entry historyEntry) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		dbRow, err := dbQ.GetEntry(ctx, db.GetEntryParams{
			Domain:   entry.payload.Domain,
			Username: entry.payload.Username,
		})
		if err != nil {
			return restoreUpdateMsg{err: err}
		}
		_, err = dbQ.UpdateEntry(ctx, db.UpdateEntryParams{
			Domain:            entry.payload.Domain,
			Username:          entry.payload.Username,
			EncryptedPassword: entry.payload.EncryptedPassword,
		})
		if err != nil {
			return restoreUpdateMsg{err: err}
		}
		entry.payload = dbRow
		return restoreUpdateMsg{data: entry}
	}
}
