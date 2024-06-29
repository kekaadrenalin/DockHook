package types

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	myErrors "github.com/kekaadrenalin/dockhook/pkg/errors"
	ociSpec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
)

type DockerCLI interface {
	ContainerList(context.Context, container.ListOptions) ([]types.Container, error)
	ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error)
	Events(context.Context, events.ListOptions) (<-chan events.Message, <-chan error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error)
	Ping(ctx context.Context) (types.Ping, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ociSpec.Platform, containerName string) (container.CreateResponse, error)
	Info(ctx context.Context) (system.Info, error)
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
}

type Client interface {
	ListContainers() ([]Container, error)
	FindContainerByID(string) (Container, error)
	ContainerLogs(context.Context, string, *time.Time, StdType) (io.ReadCloser, error)
	Events(context.Context, chan<- ContainerEvent) error
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time, StdType) (io.ReadCloser, error)
	Ping(context.Context) (types.Ping, error)
	Host() *Host
	ContainerActions(webhook *Webhook) (*Container, *myErrors.HTTPError)
	TryImagePull(imageRef string, registryAuth string) (bool, error)
	IsSwarmMode() bool
	SystemInfo() system.Info
}

type StdType int

const (
	UNKNOWN StdType = 1 << iota
	STDOUT
	STDERR
)
const STDALL = STDOUT | STDERR

func (s StdType) String() string {
	switch s {
	case STDOUT:
		return "stdout"
	case STDERR:
		return "stderr"
	case STDALL:
		return "all"
	default:
		return "unknown"
	}
}

type Host struct {
	Name       string   `json:"name"`
	ID         string   `json:"id"`
	URL        *url.URL `json:"-"`
	CertPath   string   `json:"-"`
	CACertPath string   `json:"-"`
	KeyPath    string   `json:"-"`
	ValidCerts bool     `json:"-"`
	NCPU       int      `json:"nCPU"`
	MemTotal   int64    `json:"memTotal"`
}

func (h *Host) GetDescription() string {
	return fmt.Sprintf("ID: %s --> Name: %s", h.ID, h.Name)
}
