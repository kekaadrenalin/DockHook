package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/network"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	myErrors "github.com/kekaadrenalin/dockhook/pkg/errors"
	myTypes "github.com/kekaadrenalin/dockhook/pkg/types"
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

type httpClient struct {
	cli     myTypes.DockerCLI
	filters filters.Args
	host    *myTypes.Host
	info    system.Info
}

func NewClient(cli myTypes.DockerCLI, filters filters.Args, host *myTypes.Host) myTypes.Client {
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
func NewClientWithFilters(f map[string][]string) (myTypes.Client, error) {
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

	return NewClient(cli, filterArgs, &myTypes.Host{Name: "localhost", ID: "localhost"}), nil
}

// NewClientWithTLSAndFilter creates a new instance of Client with docker filters for remote hosts
func NewClientWithTLSAndFilter(f map[string][]string, host myTypes.Host) (myTypes.Client, error) {
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

// FindContainerByID finds a container by ID
func (d *httpClient) FindContainerByID(id string) (myTypes.Container, error) {
	var containerItem myTypes.Container
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

// FindContainerByName finds a container by Name
func (d *httpClient) FindContainerByName(name string) (myTypes.Container, error) {
	var containerItem myTypes.Container
	containers, err := d.ListContainers()
	if err != nil {
		return containerItem, err
	}

	found := false
	for _, c := range containers {
		if c.Name == name {
			containerItem = c
			found = true
			break
		}
	}
	if !found {
		return containerItem, fmt.Errorf("unable to find containerItem with name: %s", name)
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

func (d *httpClient) ContainerActions(webhook *myTypes.Webhook) (*myTypes.Container, *myErrors.HTTPError) {
	var err error
	var containerItem myTypes.Container

	if webhook.Action == myTypes.Action.PULL {
		containerItem, err = d.FindContainerByName(webhook.ContainerName)
	} else {
		containerItem, err = d.FindContainerByID(webhook.ContainerId)
	}
	if err != nil {
		return nil, &myErrors.HTTPError{
			Err:        err,
			StatusCode: http.StatusNotFound,
			Message:    fmt.Sprintf("no container found %s", webhook.ContainerId),
		}
	}

	err = func() error {
		switch webhook.Action {
		case myTypes.Action.START:
			return d.StartContainer(context.Background(), containerItem.ID)

		case myTypes.Action.STOP:
			return d.StopContainer(context.Background(), containerItem.ID)

		case myTypes.Action.RESTART:
			return d.RestartContainer(context.Background(), containerItem.ID)

		case myTypes.Action.PULL:
			return d.PullAndRestartContainer(context.Background(), webhook, containerItem)

		default:
			return fmt.Errorf("unknown action: %s", webhook.Action)
		}
	}()
	if err != nil {
		return nil, &myErrors.HTTPError{
			Err:        err,
			StatusCode: http.StatusInternalServerError,
		}
	}

	return &containerItem, nil
}

// ListContainers lists all containers
func (d *httpClient) ListContainers() ([]myTypes.Container, error) {
	containerListOptions := container.ListOptions{
		Filters: d.filters,
		All:     true,
	}
	list, err := d.cli.ContainerList(context.Background(), containerListOptions)
	if err != nil {
		return nil, err
	}

	var containers = make([]myTypes.Container, 0, len(list))
	for _, c := range list {
		name := "no name"
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		containerItem := myTypes.Container{
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

func (d *httpClient) ContainerLogs(ctx context.Context, id string, since *time.Time, stdType myTypes.StdType) (io.ReadCloser, error) {
	log.WithField("id", id).WithField("since", since).WithField("stdType", stdType).Debug("streaming logs for container")

	sinceQuery := ""
	if since != nil {
		sinceQuery = since.Add(time.Millisecond).Format(time.RFC3339Nano)
	}

	options := container.LogsOptions{
		ShowStdout: stdType&myTypes.STDOUT != 0,
		ShowStderr: stdType&myTypes.STDERR != 0,
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

func (d *httpClient) Events(ctx context.Context, messages chan<- myTypes.ContainerEvent) error {
	dockerMessages, err := d.cli.Events(ctx, events.ListOptions{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-err:
			return err

		case message := <-dockerMessages:
			if message.Type == events.ContainerEventType && len(message.Actor.ID) > 0 {
				messages <- myTypes.ContainerEvent{
					ActorID: message.Actor.ID[:12],
					Name:    string(message.Action),
					Host:    d.host.ID,
				}
			}
		}
	}

}

func (d *httpClient) ContainerLogsBetweenDates(ctx context.Context, id string, from time.Time, to time.Time, stdType myTypes.StdType) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: stdType&myTypes.STDOUT != 0,
		ShowStderr: stdType&myTypes.STDERR != 0,
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

func (d *httpClient) Host() *myTypes.Host {
	return d.host
}

func (d *httpClient) IsSwarmMode() bool {
	return d.info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive
}

func (d *httpClient) SystemInfo() system.Info {
	return d.info
}

// PullLatestImage pulls the latest version of an image
func (d *httpClient) PullLatestImage(ctx context.Context, imageName string, registryAuth string) error {
	log.Debugf("Pulling latest image for %s", imageName)

	out, err := d.cli.ImagePull(ctx, imageName, image.PullOptions{RegistryAuth: registryAuth})
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
func (d *httpClient) PullAndRestartContainer(ctx context.Context, webhook *myTypes.Webhook, containerItem myTypes.Container) error {
	containerID := containerItem.ID

	containerInspect, err := d.cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return err
	}

	imageName := containerInspect.Config.Image
	if err := d.PullLatestImage(ctx, imageName, webhook.Auth); err != nil {
		return err
	}

	config := containerInspect.Config
	hostConfig := containerInspect.HostConfig
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: containerInspect.NetworkSettings.Networks,
	}

	if err = d.cli.ContainerStop(context.Background(), containerID, container.StopOptions{}); err != nil {
		return err
	}

	log.Debugf("Stoped Container ID: %s\n", containerID)

	if err = d.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{}); err != nil {
		return err
	}

	log.Debugf("Removed Container ID: %s\n", containerID)

	newContainer, err := d.cli.ContainerCreate(context.Background(), config, hostConfig, networkingConfig, nil, containerInspect.Name)
	if err != nil {
		log.Fatalf("Error creating container: %v", err)
	}

	log.Debugf("Created Container ID: %s\n", newContainer.ID)

	return d.cli.ContainerStart(context.Background(), newContainer.ID, container.StartOptions{})
}

func (d *httpClient) TryImagePull(imageName string, registryAuth string) (bool, error) {
	_, err := d.cli.ImagePull(context.Background(), imageName, image.PullOptions{RegistryAuth: registryAuth})
	if err != nil {
		log.Debugf("err: %T %+v\n", err, err)

		return false, err
	}

	return true, nil
}

var ParenthesisRe = regexp.MustCompile(`\(([a-zA-Z]+)\)`)

func findBetweenParentheses(s string) string {
	if results := ParenthesisRe.FindStringSubmatch(s); results != nil {
		return results[1]
	}

	return ""
}
