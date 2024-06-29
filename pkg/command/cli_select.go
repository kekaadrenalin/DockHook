package command

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"

	"github.com/kekaadrenalin/dockhook/pkg/types"
)

type selectItems []selectItem
type selectItem struct {
	uuid  string
	title string
}
type cliSelectModel struct {
	cursor int
	choice string
}

var cliSelectChoices = selectItems{}
var (
	ok     bool
	p      *tea.Program
	result cliSelectModel
)

var cliSelectTitle = "Select your choice:\n\n"

func populateChoicesWithClients(clients map[string]types.Client) map[string]types.Client {
	clearSelectChoices("Select a client:\n\n")
	storeClients := map[string]types.Client{}

	for host, client := range clients {
		storeClients[host] = client
		cliSelectChoices = append(cliSelectChoices, selectItem{uuid: host, title: client.Host().GetDescription()})
	}

	return storeClients
}

func populateChoicesWithContainers(containers []types.Container) map[string]types.Container {
	clearSelectChoices("Select a container:\n\n")
	storeContainers := map[string]types.Container{}

	for _, container := range containers {
		storeContainers[container.ID] = container
		cliSelectChoices = append(cliSelectChoices, selectItem{uuid: container.ID, title: container.GetDescription()})
	}

	return storeContainers
}

func selectChoice() string {
	p = tea.NewProgram(cliSelectModel{})

	m, err := p.Run()
	if err != nil {
		log.Fatalf("Unknown error: %s\n", err)
	}

	if result, ok = m.(cliSelectModel); !ok || result.choice == "" {
		log.Infoln("Good buy! :)")
		os.Exit(0)
	}

	return result.choice
}

func (m cliSelectModel) Init() tea.Cmd { return nil }
func (m cliSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// Send the choice on the channel and exit.
			m.choice = cliSelectChoices[m.cursor].uuid
			return m, tea.Quit

		case "down", "j", "s":
			m.cursor++
			if m.cursor >= len(cliSelectChoices) {
				m.cursor = 0
			}

		case "up", "k", "w":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(cliSelectChoices) - 1
			}
		}
	}

	return m, nil
}
func (m cliSelectModel) View() string {
	s := strings.Builder{}
	s.WriteString(cliSelectTitle)

	for i := 0; i < len(cliSelectChoices); i++ {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(cliSelectChoices[i].title)
		s.WriteString("\n")
	}

	s.WriteString("\n(press q to quit)\n")

	return s.String()
}

func clearSelectChoices(newChoiceTitle string) {
	cliSelectChoices = selectItems{}
	cliSelectTitle = newChoiceTitle
}
