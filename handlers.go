package main

import (
	"database/sql"
	"fmt"

	"charm.land/bubbletea/v2"
)

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
func (m model) handleRegUpdate(msg regUpdateMsg) (model, tea.Cmd) {
	if msg.err != nil {
		if msg.err != sql.ErrNoRows {
			m.status.text = msg.err.Error()
			m.status.kind = statusError
			return m, nil
		}
		m.status.text = "entry already exists. Do you want to update the password?"
	}
	m.status.kind = statusNone
	m.updateBuffer = updateBuffer{
		payload:  msg.data.entry,
		password: msg.data.password,
	}
	m.updateConfirmation = true
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
