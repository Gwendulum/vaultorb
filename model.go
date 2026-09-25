package main

import (
	"context"
	"fmt"
	"strings"
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

type item interface {
	Title() string
}

type menuItem string

func (m menuItem) Title() string {
	return string(m)
}

type listEntry struct {
	row db.ListEntriesRow
}

func (l listEntry) Title() string {
	return l.row.Domain + " " + l.row.Username
}

type model struct {
	activeScreen  currentScreen
	masterKey     []byte
	choices       []item
	cursor        int
	loginInput    textinput.Model
	domainInput   textinput.Model
	usernameInput textinput.Model
	passwordInput textinput.Model
	focusIndex    int
	svc           services
	password      string
	status        statusMessage
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
		loginInput:    login,
		domainInput:   dom,
		usernameInput: usr,
		passwordInput: pass,
		focusIndex:    0,
		svc:           svc,
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

type regMsg cmdMsg[string]

func registerCmd(dbQ *db.Queries, domUserPass string, masterKey []byte) tea.Cmd {
	return func() tea.Msg {
		splitString := strings.Fields(domUserPass)
		ctx := context.Background()
		msg, err := registerPassword(ctx, dbQ, splitString, masterKey)
		if err != nil {
			return regMsg{err: err}
		}
		return regMsg{data: msg}
	}
}

type listMsg cmdMsg[[]db.ListEntriesRow]

func listCmd(dbQ *db.Queries) tea.Cmd {
	return func() tea.Msg {
		list, err := listPassword(context.Background(), dbQ)
		if err != nil {
			return listMsg{err: err}
		}
		return listMsg{data: list}
	}
}

type deleteMsg cmdMsg[string]

func deleteCmd(dbQ *db.Queries, domUser string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		splitString := strings.Fields(domUser)
		err := deletePassword(ctx, dbQ, splitString)
		if err != nil {
			return deleteMsg{err: err}
		}
		return deleteMsg{data: "Deletion successful!"}

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

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case authMsg:
		return m.handleAuth(msg)
	case getMsg:
		return m.handleGet(msg)
	case regMsg:
		return m.handleReg(msg)
	case listMsg:
		return m.handleList(msg)
	case deleteMsg:
		return m.handleDelete(msg)
	case generateMsg:
		return m.handleGenerate(msg)
	}
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.activeScreen {

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

	case screenList:
		s += "list\n"

		for i, entry := range m.choices {
			if m.cursor == i {
				s += "|> | "

			} else {
				s += "   | "
			}

			s += fmt.Sprintf("%s\n", entry.Title())
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
	m.status.text = fmt.Sprintf("\nSuccessfully registered %s", msg.data)
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
	m.status.text = msg.data
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

func (m model) switchScreen(screen currentScreen) (model, tea.Cmd) {
	m.loginInput.Reset()
	m.domainInput.Reset()
	m.usernameInput.Reset()
	m.domainInput.Blur()
	m.usernameInput.Blur()

	m.cursor = 0
	m.focusIndex = 0
	m.activeScreen = screen
	m.status.kind = statusNone
	m.status.text = ""

	var cmd tea.Cmd
	switch screen {
	case screenLogin:
		cmd = m.loginInput.Focus()
	case screenDashboard:
		m.choices = []item{
			menuItem("register"),
			menuItem("get"),
			menuItem("list"),
			menuItem("delete")}

	case screenGet, screenRegister, screenDelete:
		cmd = m.domainInput.Focus()
	}

	return m, cmd
}

func (m model) updateScreenLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "enter" {
			password := m.loginInput.Value()
			m.loginInput.Reset()
			return m, authenticateCmd(m.svc.dbQ, password)
		}
	}

	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

func (m model) updateScreenDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {

		switch keyMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {

				m.cursor++
			}
		case "space", "enter":
			switch m.cursor {
			case 0:
				return m.switchScreen(screenRegister)
			case 1:
				return m.switchScreen(screenGet)
			case 2:
				m, _ = m.switchScreen(screenList)
				return m, listCmd(m.svc.dbQ)
			case 3:
				return m.switchScreen(screenDelete)
			}
		}
	}
	return m, cmd
}

func (m model) updateScreenGet(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		m.status.text = ""
		m.status.kind = statusNone
		switch keyMsg.String() {
		case "enter":
			getPassword := m.domainInput.Value() + " " + m.usernameInput.Value()
			return m, getCmd(m.svc.dbQ, getPassword, m.masterKey)
		case "esc":
			m, cmd := m.switchScreen(screenDashboard)
			return m, cmd

		case "tab":
			switch m.focusIndex {
			case 0:
				m.focusIndex = 1
				m.domainInput.Blur()
				cmd = m.usernameInput.Focus()
			case 1:
				m.focusIndex = 0
				m.usernameInput.Blur()
				cmd = m.domainInput.Focus()
			}
			return m, cmd
		}
	}
	switch m.focusIndex {
	case 0:

		m.domainInput, cmd = m.domainInput.Update(msg)
	case 1:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	}
	return m, cmd
}

func (m model) updateScreenRegister(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		m.status.text = ""
		m.status.kind = statusNone
		switch keyMsg.String() {
		case "enter":
			regPassword := m.domainInput.Value() + " " + m.usernameInput.Value() + " " + m.passwordInput.Value()
			return m, registerCmd(m.svc.dbQ, regPassword, m.masterKey)

		case "esc":
			return m.switchScreen(screenDashboard)

		case "tab":
			switch m.focusIndex {
			case 0:
				m.focusIndex = 1

				m.domainInput.Blur()
				cmds = append(cmds, m.usernameInput.Focus())
				m.passwordInput.Blur()
			case 1:
				m.focusIndex = 2

				m.domainInput.Blur()
				m.usernameInput.Blur()
				cmds = append(cmds, m.passwordInput.Focus())
			case 2:
				m.focusIndex = 0

				cmds = append(cmds, m.domainInput.Focus())
				m.usernameInput.Blur()
				m.passwordInput.Blur()
			}

		case "ctrl+g":
			cmds = append(cmds, generateCmd(m.svc))
		}
	}
	var inputCmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.domainInput, inputCmd = m.domainInput.Update(msg)
	case 1:
		m.usernameInput, inputCmd = m.usernameInput.Update(msg)

	case 2:
		m.passwordInput, inputCmd = m.passwordInput.Update(msg)

	}
	cmds = append(cmds, inputCmd)

	return m, tea.Batch(cmds...)

}

func (m model) updateScreenList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		m.status.text = ""
		m.status.kind = statusNone
		switch keyMsg.String() {

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "esc":
			m, cmd := m.switchScreen(screenDashboard)
			return m, cmd

		case "enter":
			if entry, ok := m.ChoiceValidation(); ok {
				return m, getCmd(m.svc.dbQ, entry.Title(), m.masterKey)
			}
		case "ctrl+d":
			if entry, ok := m.ChoiceValidation(); ok {
				return m, deleteCmd(m.svc.dbQ, entry.Title())
			}
		}
	}
	return m, cmd
}

func (m model) updateScreenDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		m.status.text = ""
		m.status.kind = statusNone
		switch keyMsg.String() {

		case "enter":
			getDeletion := m.domainInput.Value() + " " + m.usernameInput.Value()
			return m, deleteCmd(m.svc.dbQ, getDeletion)

		case "esc":
			m, cmd := m.switchScreen(screenDashboard)
			return m, cmd
		case "tab":
			switch m.focusIndex {
			case 0:
				m.focusIndex = 1
				m.domainInput.Blur()
				cmd = m.usernameInput.Focus()
			case 1:
				m.focusIndex = 0
				m.usernameInput.Blur()
				cmd = m.domainInput.Focus()
			}
		}
	}
	switch m.focusIndex {
	case 0:

		m.domainInput, cmd = m.domainInput.Update(msg)
	case 1:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	}
	return m, cmd
}

func (m model) ChoiceValidation() (item, bool) {
	if m.cursor < 0 || m.cursor >= len(m.choices) {
		m.status.text = "Invalid selection"
		m.status.kind = statusError
		return nil, false
	}
	m.status.text = ""
	m.status.kind = statusNone
	return m.choices[m.cursor], true
}
