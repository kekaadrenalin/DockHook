package webhook

import (
	"os"
	"testing"
	"time"

	"github.com/kekaadrenalin/dockhook/pkg/types"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func Test_Webhooks_ReadFromFile_happy(t *testing.T) {
	testFile := "test_webhooks.yaml"
	defer os.Remove(testFile)

	expectedWebhooks := WebhooksDatabase{
		Webhooks: map[string]*types.Webhook{
			"uuid1": {
				UUID:          "uuid1",
				ContainerId:   "container1",
				ContainerName: "containerName1",
				Host:          "host1",
				Action:        "start",
				Created:       time.Now(),
			},
		},
	}

	err := setupTestFile(testFile, expectedWebhooks)
	assert.NoError(t, err)

	webhooksDB, err := ReadWebhooksFromFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, expectedWebhooks.Webhooks["uuid1"].UUID, webhooksDB.Webhooks["uuid1"].UUID)
}

func Test_Webhooks_ReadFromFile_error_exist(t *testing.T) {
	testFile := "non_existent.yaml"

	webhooksDB, err := ReadWebhooksFromFile(testFile)
	assert.NoError(t, err)
	assert.Empty(t, webhooksDB.Webhooks)
}

func Test_Webhooks_Create_happy(t *testing.T) {
	testFile := "test_create_webhook.yaml"
	defer os.Remove(testFile)

	webhook := types.Webhook{
		UUID:          "uuid2",
		ContainerId:   "container2",
		ContainerName: "containerName2",
		Host:          "host2",
		Action:        "stop",
		Created:       time.Now(),
	}

	createdWebhook, err := CreateWebhook(testFile, webhook)
	assert.NoError(t, err)
	assert.Equal(t, webhook.UUID, createdWebhook.UUID)

	webhooksDB, err := ReadWebhooksFromFile(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, webhooksDB.Webhooks[webhook.UUID])
	assert.Equal(t, webhook.UUID, webhooksDB.Webhooks[webhook.UUID].UUID)
}

func Test_Webhooks_Create_error_exists(t *testing.T) {
	testFile := "test_create_webhook_exists.yaml"
	defer os.Remove(testFile)

	webhooksDB := WebhooksDatabase{
		Webhooks: map[string]*types.Webhook{
			"uuid3": {
				UUID:          "uuid3",
				ContainerId:   "container3",
				ContainerName: "containerName3",
				Host:          "host3",
				Action:        "restart",
				Created:       time.Now(),
			},
		},
	}
	err := setupTestFile(testFile, webhooksDB)
	assert.NoError(t, err)

	webhook := types.Webhook{
		UUID:          "uuid3",
		ContainerId:   "container3",
		ContainerName: "containerName3",
		Host:          "host3",
		Action:        "restart",
		Created:       time.Now(),
	}

	_, err = CreateWebhook(testFile, webhook)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook uuid3 is exists")
}

func setupTestFile(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	return encoder.Encode(data)
}
