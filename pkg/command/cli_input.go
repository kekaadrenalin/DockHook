package command

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"

	"github.com/charmbracelet/bubbles/textinput"
)

type (
	inputErrMsg error
)

type cliInputModel struct {
	textInput textinput.Model
	err       error
}

var cliInputTitle = "Input your data"

func NewCliInput(placeholder string, newChoiceTitle string, charLimit int) string {
	p := tea.NewProgram(initInputChoice(placeholder, newChoiceTitle, charLimit))

	m, err := p.Run()
	if err != nil {
		log.Fatalf("Unknown error: %s\n", err)
	}

	result, ok := m.(cliInputModel)
	if !ok || result.textInput.Value() == "" {
		log.Infoln("Good buy! :)")
		os.Exit(0)
	}

	return result.textInput.Value()
}

func initInputChoice(placeholder string, newChoiceTitle string, charLimit int) cliInputModel {
	cliInputTitle = newChoiceTitle

	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = charLimit
	ti.Width = 40

	return cliInputModel{
		textInput: ti,
		err:       nil,
	}
}

func (m cliInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m cliInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case inputErrMsg:
		m.err = msg

		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m cliInputModel) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		cliInputTitle,
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"
}
