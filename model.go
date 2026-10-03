package main

import (
	"fmt"
	"vaultorb/internal/db"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbletea/v2"
)

type currentScreen int

const (
	screenLogin currentScreen = iota
	screenDashboard
	screenRegister
	screenGet
	screenList
	screenDelete
	screenCreateMasterPassword
)

type statusType int

const (
	statusNone statusType = iota
	statusSuccess
	statusError
)

type statusMessage struct {
	text string
	kind statusType
}

type undoActionType int

const (
	undoActionRegister = iota
	undoActionUpdate
	undoActionDelete
)

type item interface {
	Title() string
}

type menuItem string

func (m menuItem) Title() string {
	return string(m)
}

type listEntry struct {
	row db.Entry
}

func (l listEntry) Title() string {
	return l.row.Domain + " " + l.row.Username
}

type historyEntry struct {
	Kind    undoActionType
	payload db.Entry
	isUndo  bool
}

type updateBuffer struct {
	payload  db.Entry
	password string
}

type model struct {
	activeScreen       currentScreen
	masterKey          []byte
	choices            []item
	cursor             int
	editingIndex       int
	loginInput         textinput.Model
	domainInput        textinput.Model
	usernameInput      textinput.Model
	passwordInput      textinput.Model
	focusIndex         int
	svc                services
	password           string
	status             statusMessage
	undoHistory        []historyEntry
	redoHistory        []historyEntry
	updateBuffer       updateBuffer
	updateConfirmation bool
}

func initialModel(svc services) model {
	login := textinput.New()
	login.Placeholder = "Enter Master Password"
	login.EchoMode = textinput.EchoPassword
	login.Focus()

	dom := textinput.New()
	dom.Placeholder = "Domain"

	usr := textinput.New()
	usr.Placeholder = "Username"

	pass := textinput.New()
	pass.Placeholder = "Password"
	return model{
		activeScreen:  screenLogin,
		masterKey:     nil,
		cursor:        0,
		editingIndex:  -1,
		loginInput:    login,
		domainInput:   dom,
		usernameInput: usr,
		passwordInput: pass,
		focusIndex:    0,
		svc:           svc,
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textinput.Blink)
	cmds = append(cmds, checkInitialRunCmd(m.svc.dbQ))
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case checkInitialRunMsg:
		return m.handleCheckInitialRun(msg)
	case authMsg:
		return m.handleAuth(msg)
	case getMsg:
		return m.handleGet(msg)
	case regMsg:
		return m.handleReg(msg)
	case regUpdateMsg:
		return m.handleRegUpdate(msg)
	case listMsg:
		return m.handleList(msg)
	case deleteMsg:
		return m.handleDelete(msg)
	case updateMsg:
		return m.handleUpdate(msg)
	case generateMsg:
		return m.handleGenerate(msg)
	case restoreMsg:
		return m.handleRestore(msg)
	case redeleteMsg:
		return m.handleRedelete(msg)
	case restoreUpdateMsg:
		return m.handleRestoreUpdate(msg)
	}
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && (keyMsg.String() == "ctrl+z" || keyMsg.String() == "ctrl+r") {
		var entry historyEntry
		var err error
		switch keyMsg.String() {
		case "ctrl+z":
			m, entry, err = m.popUndo()
		case "ctrl+r":
			m, entry, err = m.popRedo()
		}
		if err != nil {
			m.status.kind = statusError
			m.status.text = err.Error()
			return m, nil
		}
		switch entry.Kind {
		case undoActionDelete:
			return m, restoreCmd(m.svc.dbQ, entry)
		case undoActionRegister:
			return m, redeleteCmd(m.svc.dbQ, entry)
		case undoActionUpdate:
			return m, restoreUpdateCmd(m.svc.dbQ, entry)
		}
	}
	switch m.activeScreen {

	case screenCreateMasterPassword:
		return m.updateScreenCreateMasterPassword(msg)

	case screenLogin:
		return m.updateScreenLogin(msg)

	case screenDashboard:
		return m.updateScreenDashboard(msg)
	case screenGet:
		return m.updateScreenGet(msg)

	case screenRegister:
		return m.updateScreenRegister(msg)

	case screenList:
		return m.updateScreenList(msg)

	case screenDelete:
		return m.updateScreenDelete(msg)
	}

	return m, cmd
}

func (m model) View() tea.View {
	s := "VaultOrb CLI\n\n"

	switch m.activeScreen {
	case screenCreateMasterPassword:

		if m.status.kind == statusError {
			s += "something went wrong\n\n"
		}
		s += "Create your master password\n"
		s += fmt.Sprintf("%v", m.loginInput.View())

	case screenLogin:
		if m.status.kind == statusError {
			s += "wrong password\n\n"
		}
		s += "login\n"
		s += fmt.Sprintf("%v", m.loginInput.View())

	case screenDashboard:

		s += "dashboard\n\n"

		for i, choice := range m.choices {
			if m.cursor == i {
				s += "|>| "
			} else {
				s += "  | "
			}

			s += fmt.Sprintf("%s\n", choice.Title())
		}
	case screenGet:
		s += "Get\n"
		s += fmt.Sprintf("|%v |%v\n\n", m.domainInput.View(), m.usernameInput.View())
		if m.status.kind == statusError {
			s += fmt.Sprintf("error: %v", m.status.text)
		}
		if m.status.kind == statusSuccess {
			s += m.password
		}

	case screenRegister:
		s += "Register\n"
		s += fmt.Sprintf("%v %v %v\n\n", m.domainInput.View(), m.usernameInput.View(), m.passwordInput.View())
		if m.status.kind == statusError {
			s += fmt.Sprintf("error: %v", m.status.text)
		}
		s += m.status.text + "\n"
		if m.updateConfirmation {
			s += "press enter to update to new password or escape to cancel\n"
		}

	case screenList:
		s += "list\n"

		for i, entry := range m.choices {
			if m.cursor == i {
				s += "|> | "

			} else {
				s += "   | "
			}

			s += fmt.Sprintf("%s", entry.Title())
			if m.editingIndex == i {
				s += m.passwordInput.View()
			}
			s += "\n"
		}
		switch m.status.kind {
		case statusSuccess:
			s += "\n" + m.status.text + "\n"
		case statusError:
			s += "\n" + "error: " + m.status.text + "\n"
		}

	case screenDelete:
		s += "delete\n"

		s += fmt.Sprintf("%v %v\n\n", m.domainInput.View(), m.usernameInput.View())
		if m.status.kind == statusError {
			s += fmt.Sprintf("error: %v", m.status.kind)
		}
		s += m.status.text
		s += "\n"
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

// helper functions
func (m model) popUndo() (model, historyEntry, error) {
	n := len(m.undoHistory)
	if n <= 0 {
		return m, historyEntry{}, fmt.Errorf("nothing to undo")
	}
	entry := m.undoHistory[n-1]

	m.undoHistory = m.undoHistory[:len(m.undoHistory)-1]
	if !entry.isUndo {
		return m, historyEntry{}, fmt.Errorf("redo entry on undo stack")
	}

	return m, entry, nil
}

func (m model) popRedo() (model, historyEntry, error) {
	n := len(m.redoHistory)
	if n <= 0 {
		return m, historyEntry{}, fmt.Errorf("nothing to redo")
	}
	entry := m.redoHistory[n-1]

	m.redoHistory = m.redoHistory[:len(m.redoHistory)-1]
	if entry.isUndo {
		return m, historyEntry{}, fmt.Errorf("undo entry on redo stack")
	}
	return m, entry, nil
}
