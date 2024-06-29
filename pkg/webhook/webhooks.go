package webhook

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/kekaadrenalin/dockhook/pkg/helper"
	"github.com/kekaadrenalin/dockhook/pkg/types"
	"gopkg.in/yaml.v3"
)

type WebhooksDatabase struct {
	Webhooks map[string]*types.Webhook `yaml:"webhooks"`
	LastRead time.Time                 `yaml:"-"`
	LastSave time.Time                 `yaml:"-"`
	Path     string                    `yaml:"-"`
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

func CreateWebhook(path string, webhookItem types.Webhook) (types.Webhook, error) {
	webhooks, err := ReadWebhooksFromFile(path)
	if err != nil {
		return webhookItem, err
	}

	if webhooks.Webhooks[webhookItem.UUID] != nil {
		return webhookItem, fmt.Errorf("webhook %s is exists", webhookItem.UUID)
	}

	webhooks.Webhooks[webhookItem.UUID] = &webhookItem

	if _, err := saveWebhooksToFile(webhooks, path); err != nil {
		return webhookItem, err
	}

	return webhookItem, nil
}

func saveWebhooksToFile(webhooks WebhooksDatabase, path string) (WebhooksDatabase, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, err = helper.CreateDir(path); err != nil {
			return webhooks, err
		}
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
		webhooks.Webhooks = map[string]*types.Webhook{}

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

func (d *WebhooksDatabase) Find(Uuid string) *types.Webhook {
	if err := d.readFileIfChanged(); err != nil {
		log.Errorf("Error reading users file: %s", err)
	}

	user, ok := d.Webhooks[Uuid]
	if !ok {
		return nil
	}

	return user
}
