package types

import (
	"time"
)

type Webhook struct {
	UUID          string          `json:"uuid" yaml:"-"`
	ContainerId   string          `json:"containerId" yaml:"containerId"`
	ContainerName string          `json:"containerName" yaml:"containerName"`
	Host          string          `json:"host,omitempty" yaml:"host"`
	Action        ContainerAction `json:"action" yaml:"action"`
	Auth          string          `json:"auth" yaml:"auth"`
	Created       time.Time       `json:"created" yaml:"created"`
}
