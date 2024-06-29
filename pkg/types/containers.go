package types

import (
	"fmt"
	"time"
)

// Container represents an internal representation of docker containers
type Container struct {
	ID        string            `json:"id"`
	Names     []string          `json:"names"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	ImageID   string            `json:"imageId"`
	Command   string            `json:"command"`
	Created   time.Time         `json:"created"`
	StartedAt *time.Time        `json:"startedAt,omitempty"`
	State     string            `json:"state"`
	Status    string            `json:"status"`
	Health    string            `json:"health,omitempty"`
	Host      string            `json:"host,omitempty"`
	Tty       bool              `json:"-"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ContainerEvent represents events that are triggered
type ContainerEvent struct {
	ActorID string `json:"actorId"`
	Name    string `json:"name"`
	Host    string `json:"host"`
}

type ContainerAction string

const (
	ActionStart   ContainerAction = "start"
	ActionStop    ContainerAction = "stop"
	ActionRestart ContainerAction = "restart"
	ActionPull    ContainerAction = "pull"
)

var ContainerActions = []ContainerAction{
	ActionStart,
	ActionStop,
	ActionRestart,
	ActionPull,
}

type Actions struct {
	START   ContainerAction
	STOP    ContainerAction
	RESTART ContainerAction
	PULL    ContainerAction
}

var Action = Actions{
	START:   ActionStart,
	STOP:    ActionStop,
	RESTART: ActionRestart,
	PULL:    ActionPull,
}

func (c *Container) GetDescription() string {
	return fmt.Sprintf("Id: %s --> Name: %s", c.ID, c.Name)
}

func (c *Container) GetDescriptionFull() string {
	return fmt.Sprintf("Id: %s --> Name: %s --> Image: %s", c.ID, c.Name, c.Image)
}
