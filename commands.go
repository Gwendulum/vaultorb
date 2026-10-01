package main

import(
	"fmt"
	"strings"
"database/sql"
"errors"
"context"
	"vaultorb/internal/db"
	"charm.land/bubbletea/v2"
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

func registerCmd(dbQ *db.Queries, domUserPass string, masterKey []byte) tea.Cmd {
	return func() tea.Msg {
		splitString := strings.Fields(domUserPass)
		ctx := context.Background()
		entry, err := registerPassword(ctx, dbQ, splitString, masterKey)
		if err != nil {
			return regMsg{err: err}
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

func (m model) handleCheckInitialRun(msg checkInitialRunMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	var cmd tea.Cmd
	m.status.kind = statusSuccess
	if msg.isInitialized == false {
		m, cmd = m.switchScreen(screenCreateMasterPassword)
	} else {
		m, cmd = m.switchScreen(screenLogin)
	}
	return m, cmd

}
func (m model) handleAuth(msg authMsg) (model, tea.Cmd) {
	m.loginInput.Reset()
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	m.status.kind = statusSuccess
	m.masterKey = msg.data
	m, cmd := m.switchScreen(screenDashboard)
	return m, cmd
}

func (m model) handleGet(msg getMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}

	m.status.kind = statusSuccess
	m.password = msg.data
	m.status.text = fmt.Sprintf("\nYour password is: %s", m.password)
	return m, nil
}

func (m model) handleReg(msg regMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	m.status.kind = statusSuccess
	m.status.text = fmt.Sprintf("\nSuccessfully registered %s", msg.data.Domain+": "+msg.data.Username)
	m.undoHistory = append(m.undoHistory, historyEntry{Kind: undoActionRegister, payload: msg.data, isUndo: true})
	m.redoHistory = nil
	m.domainInput.Reset()
	m.usernameInput.Reset()
	m.passwordInput.Reset()
	return m, nil
}

func (m model) handleList(msg listMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	items := make([]item, len(msg.data))
	for i, row := range msg.data {
		items[i] = listEntry{row: row}
	}
	m.choices = items
	if m.cursor < 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.choices)-1 {
		m.cursor = len(m.choices) - 1
	}
	return m, nil
}

func (m model) handleDelete(msg deleteMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	m.status.kind = statusSuccess
	m.undoHistory = append(m.undoHistory, historyEntry{Kind: undoActionDelete, payload: msg.data, isUndo: true})
	m.redoHistory = nil
	return m, listCmd(m.svc.dbQ)
}

func (m model) handleUpdate(msg updateMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	m.status.kind = statusSuccess
	m.status.text = "Password updated successfully"
	m.editingIndex = -1
	m.passwordInput.SetValue("")
	m.undoHistory = append(m.undoHistory, historyEntry{Kind: undoActionUpdate, payload: msg.data, isUndo: true})
	m.redoHistory = nil
	return m, listCmd(m.svc.dbQ)
}

func (m model) handleGenerate(msg generateMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	m.status.kind = statusSuccess
	m.passwordInput.SetValue(msg.data)
	return m, nil
}

func (m model) handleRestore(msg restoreMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	m.status.kind = statusSuccess

	if msg.data.isUndo {
		newRedoEntry := historyEntry{
			Kind:    undoActionRegister,
			payload: msg.data.payload,
			isUndo:  false,
		}
		m.redoHistory = append(m.redoHistory, newRedoEntry)
		m.status.text = "Undo: Entry restored successfully"
	} else {
		newUndoEntry := historyEntry{
			Kind:    undoActionRegister,
			payload: msg.data.payload,
			isUndo:  true,
		}
		m.undoHistory = append(m.undoHistory, newUndoEntry)
		m.status.text = "Redo: Entry restored successfully"
	}
	return m, listCmd(m.svc.dbQ)
}

func (m model) handleRedelete(msg redeleteMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	m.status.kind = statusSuccess
	if msg.data.isUndo {
		newRedoEntry := historyEntry{
			Kind:    undoActionDelete,
			payload: msg.data.payload,
			isUndo:  false,
		}
		m.redoHistory = append(m.redoHistory, newRedoEntry)
		m.status.text = "Undo: Entry redeleted successfully"
	} else {
		newUndoEntry := historyEntry{
			Kind:    undoActionDelete,
			payload: msg.data.payload,
			isUndo:  true,
		}
		m.undoHistory = append(m.undoHistory, newUndoEntry)
		m.status.text = "Redo: Entry redeleted successfully"
	}
	return m, listCmd(m.svc.dbQ)
}

func (m model) handleRestoreUpdate(msg restoreUpdateMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.status.text = msg.err.Error()
		m.status.kind = statusError
		return m, nil
	}
	if msg.data.isUndo {
		newRedoEntry := historyEntry{
			Kind:    undoActionUpdate,
			payload: msg.data.payload,
			isUndo:  false,
		}
		m.redoHistory = append(m.redoHistory, newRedoEntry)
		m.status.text = "Undo: Entry reupdated successfully"
	} else {
		newUndoEntry := historyEntry{
			Kind:    undoActionUpdate,
			payload: msg.data.payload,
			isUndo:  true,
		}
		m.undoHistory = append(m.undoHistory, newUndoEntry)
		m.status.text = "Redo: Entry reupdated successfully"
	}

	m.status.kind = statusSuccess

	return m, listCmd(m.svc.dbQ)
}
