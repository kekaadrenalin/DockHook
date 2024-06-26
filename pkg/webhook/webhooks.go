package webhook

import (
	"errors"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/kekaadrenalin/dockhook/pkg/docker"
	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"gopkg.in/yaml.v3"
)

type Webhook struct {
	UUID        string                 `json:"uuid" yaml:"-"`
	ContainerId string                 `json:"containerId" yaml:"containerId"`
	Host        string                 `json:"host,omitempty" yaml:"host"`
	Action      docker.ContainerAction `json:"action" yaml:"action"`
	Created     time.Time              `json:"created" yaml:"created"`
}

type WebhooksDatabase struct {
	Webhooks map[string]*Webhook `yaml:"webhooks"`
	LastRead time.Time           `yaml:"-"`
	LastSave time.Time           `yaml:"-"`
	Path     string              `yaml:"-"`
}

func ReadWebhooksFromFile(path string) (WebhooksDatabase, error) {
	webhooks, err := decodeWebhooksFromFile(path)
	if err != nil {
		return webhooks, err
	}

	webhooks.LastRead = time.Now()
	webhooks.Path = path

	return webhooks, nil
}

func CreateWebhook(path string, webhookItem Webhook) (Webhook, error) {
	webhooks, err := ReadWebhooksFromFile(path)
	if err != nil {
		return webhookItem, err
	}

	if webhooks.Webhooks[webhookItem.UUID] != nil {
		return webhookItem, errors.New(fmt.Sprintf("Webhook %s is exists!", webhookItem.UUID))
	}

	webhooks.Webhooks[webhookItem.UUID] = &webhookItem

	if _, err := saveWebhooksToFile(webhooks, path); err != nil {
		return webhookItem, err
	}

	return webhookItem, nil
}

func saveWebhooksToFile(webhooks WebhooksDatabase, path string) (WebhooksDatabase, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, err = helper.CreateDir(path)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return webhooks, err
	}
	defer file.Close()

	data, err := yaml.Marshal(&webhooks)
	if err != nil {
		return webhooks, err
	}

	if _, err = file.Write(data); err != nil {
		return webhooks, err
	}

	webhooks.LastSave = time.Now()

	return webhooks, nil
}

func decodeWebhooksFromFile(path string) (WebhooksDatabase, error) {
	webhooks := WebhooksDatabase{}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		webhooks.Webhooks = map[string]*Webhook{}

		return webhooks, nil
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return webhooks, err
	}
	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(&webhooks); err != nil {
		return webhooks, err
	}

	for uuid, webhook := range webhooks.Webhooks {
		webhook.UUID = uuid
	}

	return webhooks, nil
}

func (d *WebhooksDatabase) readFileIfChanged() error {
	if d.Path == "" {
		return nil
	}

	info, err := os.Stat(d.Path)
	if err != nil {
		return err
	}

	if info.ModTime().After(d.LastRead) {
		log.Infof("Found changes to %s. Updating webhooks...", d.Path)
		users, err := decodeWebhooksFromFile(d.Path)
		if err != nil {
			return err
		}
		d.Webhooks = users.Webhooks
		d.LastRead = time.Now()
	}

	return nil
}

func (d *WebhooksDatabase) Find(UUID string) *Webhook {
	if err := d.readFileIfChanged(); err != nil {
		log.Errorf("Error reading users file: %s", err)
	}

	user, ok := d.Webhooks[UUID]
	if !ok {
		return nil
	}

	return user
}
