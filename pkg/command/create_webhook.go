package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	argsType "github.com/kekaadrenalin/dockhook/pkg/types"
	log "github.com/sirupsen/logrus"

	"github.com/kekaadrenalin/dockhook/pkg/docker"
	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"github.com/kekaadrenalin/dockhook/pkg/webhook"
)

type selectItems []selectItem
type selectItem struct {
	uuid  string
	title string
}
type selectModel struct {
	cursor int
	choice string
}

var choices = selectItems{}
var (
	ok     bool
	p      *tea.Program
	result selectModel
)

func CreateWebhook(args argsType.Args) (webhook.Webhook, error) {
	path, err := filepath.Abs("./data/webhooks.yml")
	if err != nil {
		log.Fatalf("Could not find absolute path to users.yml file: %s", err)
	}

	if args.CreateWebhookCmd.DockerComposeOnly {
		args.Filter["label"] = append(args.Filter["label"], "com.docker.compose.project")
	}

	clients := docker.CreateClients(args)
	storeClients := populateChoicesWithClients(clients)
	client := storeClients[selectChoice()]

	containers, err := client.ListContainers()
	if err != nil {
		log.Fatalf("Not found containers: %s\n", err)
	}
	storeContainers := populateChoicesWithContainers(containers)
	container := storeContainers[selectChoice()]

	populateChoicesWithActions(docker.ContainerActions)
	action := docker.ContainerAction(selectChoice())

	hashData := fmt.Sprintf("%s:%s:%s", container.ID, container.Host, action)
	uuid, err := helper.GenerateUUIDv7(hashData)
	if err != nil {
		log.Fatalf("Not created UUID: %s\n", err)
	}

	return webhook.CreateWebhook(path, webhook.Webhook{
		UUID:        uuid.String(),
		ContainerId: container.ID,
		Host:        container.Host,
		Action:      action,
		Created:     time.Now(),
	})
}

func populateChoicesWithClients(clients map[string]docker.Client) map[string]docker.Client {
	clearChoices()
	storeClients := map[string]docker.Client{}

	for host, client := range clients {
		storeClients[host] = client
		choices = append(choices, selectItem{uuid: host, title: client.Host().GetDescription()})
	}

	return storeClients
}

func populateChoicesWithContainers(containers []docker.Container) map[string]docker.Container {
	clearChoices()
	storeContainers := map[string]docker.Container{}

	for _, container := range containers {
		storeContainers[container.ID] = container
		choices = append(choices, selectItem{uuid: container.ID, title: container.GetDescription()})
	}

	return storeContainers
}

func populateChoicesWithActions(actions []docker.ContainerAction) {
	clearChoices()

	for _, action := range actions {
		choices = append(choices, selectItem{uuid: string(action), title: string(action)})
	}
}

func selectChoice() string {
	p = tea.NewProgram(selectModel{})

	m, err := p.Run()
	if err != nil {
		log.Fatalf("Unknown error: %s\n", err)
	}

	if result, ok = m.(selectModel); !ok || result.choice == "" {
		log.Infoln("Good buy! :)")
		os.Exit(0)
	}

	return result.choice
}

func (m selectModel) Init() tea.Cmd { return nil }
func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// Send the choice on the channel and exit.
			m.choice = choices[m.cursor].uuid
			return m, tea.Quit

		case "down", "j", "s":
			m.cursor++
			if m.cursor >= len(choices) {
				m.cursor = 0
			}

		case "up", "k", "w":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(choices) - 1
			}
		}
	}

	return m, nil
}
func (m selectModel) View() string {
	s := strings.Builder{}
	s.WriteString("Select your choice:\n\n")

	for i := 0; i < len(choices); i++ {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(choices[i].title)
		s.WriteString("\n")
	}

	s.WriteString("\n(press q to quit)\n")

	return s.String()
}

func clearChoices() {
	choices = selectItems{}
}
