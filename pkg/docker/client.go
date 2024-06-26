package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

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

type DockerCLI interface {
	ContainerList(context.Context, container.ListOptions) ([]types.Container, error)
	ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error)
	Events(context.Context, events.ListOptions) (<-chan events.Message, <-chan error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponse, error)
	Ping(ctx context.Context) (types.Ping, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error
	Info(ctx context.Context) (system.Info, error)
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
}

type Client interface {
	ListContainers() ([]Container, error)
	FindContainer(string) (Container, error)
	ContainerLogs(context.Context, string, *time.Time, StdType) (io.ReadCloser, error)
	Events(context.Context, chan<- ContainerEvent) error
	ContainerLogsBetweenDates(context.Context, string, time.Time, time.Time, StdType) (io.ReadCloser, error)
	Ping(context.Context) (types.Ping, error)
	Host() *Host
	ContainerActions(action ContainerAction, containerID string) error
	IsSwarmMode() bool
	SystemInfo() system.Info
}

type httpClient struct {
	cli     DockerCLI
	filters filters.Args
	host    *Host
	info    system.Info
}

func NewClient(cli DockerCLI, filters filters.Args, host *Host) Client {
	clientItem := &httpClient{
		cli:     cli,
		filters: filters,
		host:    host,
	}

	var err error
	clientItem.info, err = cli.Info(context.Background())
	if err != nil {
		log.Errorf("unable to get docker info: %v", err)
	}

	host.NCPU = clientItem.info.NCPU
	host.MemTotal = clientItem.info.MemTotal

	return clientItem
}

// NewClientWithFilters creates a new instance of Client with docker filters
func NewClientWithFilters(f map[string][]string) (Client, error) {
	filterArgs := filters.NewArgs()
	for key, values := range f {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	log.Debugf("filterArgs = %v", filterArgs)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		return nil, err
	}

	return NewClient(cli, filterArgs, &Host{Name: "localhost", ID: "localhost"}), nil
}

// NewClientWithTlsAndFilter creates a new instance of Client with docker filters for remote hosts
func NewClientWithTlsAndFilter(f map[string][]string, host Host) (Client, error) {
	filterArgs := filters.NewArgs()
	for key, values := range f {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	log.Debugf("filterArgs = %v", filterArgs)

	if host.URL.Scheme != "tcp" {
		log.Fatal("Only tcp scheme is supported")
	}

	opts := []client.Opt{
		client.WithHost(host.URL.String()),
	}

	if host.ValidCerts {
		log.Debugf("Using TLS client config with certs at: %s", filepath.Dir(host.CertPath))
		opts = append(opts, client.WithTLSClientConfig(host.CACertPath, host.CertPath, host.KeyPath))
	} else {
		log.Debugf("No valid certs found, using plain TCP")
	}

	opts = append(opts, client.WithAPIVersionNegotiation())

	cli, err := client.NewClientWithOpts(opts...)

	if err != nil {
		return nil, err
	}

	return NewClient(cli, filterArgs, &host), nil
}

// FindContainer finds a container by ID
func (d *httpClient) FindContainer(id string) (Container, error) {
	var containerItem Container
	containers, err := d.ListContainers()
	if err != nil {
		return containerItem, err
	}

	found := false
	for _, c := range containers {
		if c.ID == id {
			containerItem = c
			found = true
			break
		}
	}
	if !found {
		return containerItem, fmt.Errorf("unable to find containerItem with id: %s", id)
	}

	if jsonBody, err := d.cli.ContainerInspect(context.Background(), containerItem.ID); err == nil {
		containerItem.Tty = jsonBody.Config.Tty
		if startedAt, err := time.Parse(time.RFC3339Nano, jsonBody.State.StartedAt); err == nil {
			utc := startedAt.UTC()
			containerItem.StartedAt = &utc
		}
	} else {
		return containerItem, err
	}

	return containerItem, nil
}

func (d *httpClient) ContainerActions(action ContainerAction, containerID string) error {
	switch action {
	case Action.START:
		return d.StartContainer(context.Background(), containerID)

	case Action.STOP:
		return d.StopContainer(context.Background(), containerID)

	case Action.RESTART:
		return d.RestartContainer(context.Background(), containerID)

	case Action.PULL:
		return d.PullAndRestartContainer(context.Background(), containerID)

	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// ListContainers lists all containers
func (d *httpClient) ListContainers() ([]Container, error) {
	containerListOptions := container.ListOptions{
		Filters: d.filters,
		All:     true,
	}
	list, err := d.cli.ContainerList(context.Background(), containerListOptions)
	if err != nil {
		return nil, err
	}

	var containers = make([]Container, 0, len(list))
	for _, c := range list {
		name := "no name"
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		containerItem := Container{
			ID:      c.ID[:12],
			Names:   c.Names,
			Name:    name,
			Image:   c.Image,
			ImageID: c.ImageID,
			Command: c.Command,
			Created: time.Unix(c.Created, 0),
			State:   c.State,
			Status:  c.Status,
			Host:    d.host.ID,
			Health:  findBetweenParentheses(c.Status),
			Labels:  c.Labels,
		}
		containers = append(containers, containerItem)
	}

	sort.Slice(containers, func(i, j int) bool {
		return strings.ToLower(containers[i].Name) < strings.ToLower(containers[j].Name)
	})

	return containers, nil
}

func (d *httpClient) ContainerLogs(ctx context.Context, id string, since *time.Time, stdType StdType) (io.ReadCloser, error) {
	log.WithField("id", id).WithField("since", since).WithField("stdType", stdType).Debug("streaming logs for container")

	sinceQuery := ""
	if since != nil {
		sinceQuery = since.Add(time.Millisecond).Format(time.RFC3339Nano)
	}

	options := container.LogsOptions{
		ShowStdout: stdType&STDOUT != 0,
		ShowStderr: stdType&STDERR != 0,
		Follow:     true,
		Tail:       strconv.Itoa(100),
		Timestamps: true,
		Since:      sinceQuery,
	}

	reader, err := d.cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (d *httpClient) Events(ctx context.Context, messages chan<- ContainerEvent) error {
	dockerMessages, err := d.cli.Events(ctx, events.ListOptions{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-err:
			return err

		case message := <-dockerMessages:
			if message.Type == events.ContainerEventType && len(message.Actor.ID) > 0 {
				messages <- ContainerEvent{
					ActorID: message.Actor.ID[:12],
					Name:    string(message.Action),
					Host:    d.host.ID,
				}
			}
		}
	}

}

func (d *httpClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time, stdType StdType) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: stdType&STDOUT != 0,
		ShowStderr: stdType&STDERR != 0,
		Timestamps: true,
		Since:      from.Format(time.RFC3339Nano),
		Until:      to.Format(time.RFC3339Nano),
	}

	log.Debugf("fetching logs from Docker with option: %+v", options)

	reader, err := d.cli.ContainerLogs(ctx, id, options)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (d *httpClient) Ping(ctx context.Context) (types.Ping, error) {
	return d.cli.Ping(ctx)
}

func (d *httpClient) Host() *Host {
	return d.host
}

func (d *httpClient) IsSwarmMode() bool {
	return d.info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive
}

func (d *httpClient) SystemInfo() system.Info {
	return d.info
}

// PullLatestImage pulls the latest version of an image
func (d *httpClient) PullLatestImage(ctx context.Context, imageName string) error {
	log.Debugf("Pulling latest image for %s", imageName)

	out, err := d.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(os.Stdout, out)
	return err
}

// StartContainer starts a container
func (d *httpClient) StartContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

// StopContainer stops a running container
func (d *httpClient) StopContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerStop(ctx, containerID, container.StopOptions{})
}

// RestartContainer restarts a container
func (d *httpClient) RestartContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerRestart(ctx, containerID, container.StopOptions{})
}

// PullAndRestartContainer pulls new image and restarts a container
func (d *httpClient) PullAndRestartContainer(ctx context.Context, containerID string) error {
	if containerID == "" {
		return errors.New("empty container ID")
	}

	containerInspect, err := d.cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return err
	}

	currentImage, _, err := d.cli.ImageInspectWithRaw(context.Background(), containerInspect.Config.Image)
	if err != nil {
		return err
	}

	imageName := containerInspect.Config.Image
	if err := d.PullLatestImage(ctx, imageName); err != nil {
		return err
	}

	newImage, _, err := d.cli.ImageInspectWithRaw(context.Background(), imageName)
	if err != nil {
		return err
	}

	log.Debugf("image Name: %s", imageName)
	log.Debugf("image old: %s - %s", currentImage.ID, currentImage.RepoTags)
	log.Debugf("image new: %s - %s", newImage.ID, newImage.RepoTags)

	return d.cli.ContainerRestart(context.Background(), containerID, container.StopOptions{})
}

var ParenthesisRe = regexp.MustCompile(`\(([a-zA-Z]+)\)`)

func findBetweenParentheses(s string) string {
	if results := ParenthesisRe.FindStringSubmatch(s); results != nil {
		return results[1]
	}

	return ""
}
