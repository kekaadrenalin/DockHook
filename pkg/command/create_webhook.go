package command

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types/registry"
	"github.com/goccy/go-json"
	"github.com/kekaadrenalin/dockhook/pkg/docker"
	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"github.com/kekaadrenalin/dockhook/pkg/types"
	"github.com/kekaadrenalin/dockhook/pkg/webhook"
)

func CreateWebhook(args types.Args) (types.Webhook, error) {
	path, err := filepath.Abs("./data/webhooks.yml")
	if err != nil {
		log.Fatalf("Could not find absolute path to webhooks.yml file: %s", err)
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

	populateChoicesWithActions(types.ContainerActions)
	action := types.ContainerAction(selectChoice())

	auth := getRegistryAuth(client, container.Image, action)

	hashData := fmt.Sprintf("%s:%s:%s", container.ID, container.Host, action)
	uuid, err := helper.GenerateUUIDv7(hashData)
	if err != nil {
		log.Fatalf("Not created UUID: %s\n", err)
	}

	return webhook.CreateWebhook(path, types.Webhook{
		UUID:          uuid.String(),
		ContainerId:   container.ID,
		ContainerName: container.Name,
		Host:          container.Host,
		Action:        action,
		Auth:          auth,
		Created:       time.Now(),
	})
}

func getRegistryAuth(client types.Client, imageRef string, action types.ContainerAction) string {
	auth := ""
	needAuth := false

	if action == types.ActionPull {
		storeNeedAuth := populateChoicesWithNeedAuth()
		needAuth = storeNeedAuth[selectChoice()]
	}

	if needAuth {
		username := NewCliInput("your username", "Input your username:", 100)
		password := NewCliInput("access token", "Input your access token:", 200)

		authConfig := registry.AuthConfig{
			Username: username,
			Password: password,
		}

		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			log.Fatalf("unknown err: %s", err)
		}

		auth = base64.URLEncoding.EncodeToString(encodedJSON)
	}

	success, err := client.TryImagePull(imageRef, auth)
	if err != nil || !success {
		log.Fatalf("Not valid auth: %+v, %s", success, err)
	}

	return auth
}

func populateChoicesWithActions(actions []types.ContainerAction) {
	clearSelectChoices("Select an action:\n\n")

	for _, action := range actions {
		cliSelectChoices = append(cliSelectChoices, selectItem{uuid: string(action), title: string(action)})
	}
}

func populateChoicesWithNeedAuth() map[string]bool {
	clearSelectChoices("Does the image require authorization in the registry?\n\n")
	storeNeedAuth := map[string]bool{}

	storeNeedAuth["yes"] = true
	storeNeedAuth["no"] = false

	cliSelectChoices = append(cliSelectChoices, selectItem{uuid: "yes", title: "Need AUTH"})
	cliSelectChoices = append(cliSelectChoices, selectItem{uuid: "no", title: "Dont need AUTH"})

	return storeNeedAuth
}
