package main

import (
	"charm.land/bubbletea/v2"
	"strings"
)

func (m model) updateScreenCreateMasterPassword(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	if m.editingIndex >= 0 {
		var cmds []tea.Cmd
		var inputCmd tea.Cmd
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			switch keyMsg.String() {
			case "esc":
				m.editingIndex = -1
				m.passwordInput.SetValue("")
				return m, nil
			case "ctrl+g":
				cmds = append(cmds, generateCmd(m.svc))
				return m, tea.Batch(cmds...)
			case "enter":
				if m.passwordInput.Value() == "" {
					m.status.kind = statusError
					m.status.text = "password cannot be empty"
					return m, nil
				}
				passwordValidation := strings.Fields(m.passwordInput.Value())
				if len(passwordValidation) > 1 {
					m.status.kind = statusError
					m.status.text = "password cannot have spaces"
					return m, nil
				}

				entryItem := m.choices[m.cursor]
				return m, updateCmd(m.svc.dbQ, entryItem.Title(), m.passwordInput.Value(), m.masterKey)

			}

		}
		m.passwordInput, inputCmd = m.passwordInput.Update(msg)
		cmds = append(cmds, inputCmd)
		return m, tea.Batch(cmds...)
	}
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
		case "ctrl+u":
			if _, ok := m.ChoiceValidation(); ok {
				m.editingIndex = m.cursor
				m.focusIndex = 2
				m.domainInput.Blur()
				m.usernameInput.Blur()
				cmd = m.passwordInput.Focus()
				return m, cmd
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

// helper functions
func (m model) switchScreen(screen currentScreen) (model, tea.Cmd) {
	m.loginInput.Reset()
	m.domainInput.Reset()
	m.usernameInput.Reset()
	m.passwordInput.Reset()

	m.loginInput.Blur()
	m.domainInput.Blur()
	m.usernameInput.Blur()
	m.passwordInput.Blur()

	m.cursor = 0
	m.focusIndex = 0
	m.activeScreen = screen
	m.status.kind = statusNone
	m.status.text = ""

	var cmd tea.Cmd
	switch screen {
	case screenLogin, screenCreateMasterPassword:
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
