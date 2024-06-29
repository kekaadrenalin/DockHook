package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	ociSpec "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
	"time"

	"testing"

	myTypes "github.com/kekaadrenalin/dockhook/pkg/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockedProxy struct {
	mock.Mock
	myTypes.DockerCLI
}

func (m *mockedProxy) ContainerList(context.Context, container.ListOptions) ([]types.Container, error) {
	args := m.Called()
	containers, ok := args.Get(0).([]types.Container)
	if !ok && args.Get(0) != nil {
		panic("containers is not of types []types.Container")
	}

	return containers, args.Error(1)
}

func (m *mockedProxy) ContainerLogs(ctx context.Context, id string, options container.LogsOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, id, options)
	reader, ok := args.Get(0).(io.ReadCloser)
	if !ok && args.Get(0) != nil {
		panic("reader is not of types io.ReadCloser")
	}

	return reader, args.Error(1)
}

func (m *mockedProxy) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	args := m.Called(ctx, containerID)

	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func (m *mockedProxy) ContainerStats(_ context.Context, _ string, _ bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{}, nil
}

func (m *mockedProxy) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func (m *mockedProxy) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func (m *mockedProxy) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	args := m.Called(ctx, containerID, options)
	err := args.Get(0)

	if err != nil {
		return args.Error(0)
	}

	return nil
}

func (m *mockedProxy) ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error) {
	args := m.Called(ctx, refStr, options)
	reader, ok := args.Get(0).(io.ReadCloser)
	if !ok && args.Get(0) != nil {
		panic("reader is not of types io.ReadCloser")
	}

	return reader, args.Error(1)
}

func (m *mockedProxy) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	args := m.Called(ctx, imageID)

	return args.Get(0).(types.ImageInspect), args.Get(1).([]byte), args.Error(2)
}

func (m *mockedProxy) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	args := m.Called(ctx, containerID, options)

	return args.Error(0)
}

func (m *mockedProxy) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ociSpec.Platform, containerName string) (container.CreateResponse, error) {
	args := m.Called(ctx, config, hostConfig, networkingConfig, platform, containerName)

	return args.Get(0).(container.CreateResponse), args.Error(1)
}

func Test_dockerClient_ListContainers_null(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)
	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	list, err := client.ListContainers()
	assert.Empty(t, list, "list should be empty")
	require.NoError(t, err, "error should not return an error.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	list, err := client.ListContainers()
	assert.Nil(t, list, "list should be nil")
	require.Error(t, err, "test.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_happy(t *testing.T) {
	containers := []types.Container{
		{
			ID:    "abcdefghijklmnopqrst",
			Names: []string{"/z_test_container"},
		},
		{
			ID:    "1234567890_abcxyzdef",
			Names: []string{"/a_test_container"},
		},
	}

	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	list, err := client.ListContainers()
	require.NoError(t, err, "error should not return an error.")

	Ids := []string{"1234567890_a", "abcdefghijkl"}
	for i, containerItem := range list {
		assert.Equal(t, containerItem.ID, Ids[i])
	}

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_happy(t *testing.T) {
	id := "123456"

	proxy := new(mockedProxy)
	expected := "INFO Testing logs..."
	b := make([]byte, 8)

	binary.BigEndian.PutUint32(b[4:], uint32(len(expected)))
	b = append(b, []byte(expected)...)

	reader := io.NopCloser(bytes.NewReader(b))
	since := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "100",
		Timestamps: true,
		Since:      "2021-01-01T00:00:00.001Z"}
	proxy.On("ContainerLogs", mock.Anything, id, options).Return(reader, nil)

	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}
	logReader, _ := client.ContainerLogs(context.Background(), id, &since, myTypes.STDALL)

	actual, _ := io.ReadAll(logReader)
	assert.Equal(t, string(b), string(actual), "message doesn't match expected")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_error(t *testing.T) {
	id := "123456"
	proxy := new(mockedProxy)

	proxy.On("ContainerLogs", mock.Anything, id, mock.Anything).Return(nil, errors.New("test"))

	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	reader, err := client.ContainerLogs(context.Background(), id, nil, myTypes.STDALL)

	assert.Nil(t, reader, "reader should be nil")
	assert.Error(t, err, "error should have been returned")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_FindContainer_happy(t *testing.T) {
	containers := []types.Container{
		{
			ID:    "abcdefghijklmnopqrst",
			Names: []string{"/z_test_container"},
		},
		{
			ID:    "1234567890_abcxyzdef",
			Names: []string{"/a_test_container"},
		},
	}

	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)

	state := &types.ContainerState{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}
	json := types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{State: state}, Config: &container.Config{Tty: false}}
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(json, nil)

	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	containerItem, err := client.FindContainerByID("abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")

	assert.Equal(t, containerItem.ID, "abcdefghijkl")

	proxy.AssertExpectations(t)
}
func Test_dockerClient_FindContainer_error(t *testing.T) {
	containers := []types.Container{
		{
			ID:    "abcdefghijklmnopqrst",
			Names: []string{"/z_test_container"},
		},
		{
			ID:    "1234567890_abcxyzdef",
			Names: []string{"/a_test_container"},
		},
	}

	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	_, err := client.FindContainerByID("not_valid")
	require.Error(t, err, "error should be thrown")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerActions_happy(t *testing.T) {
	containers := []types.Container{
		{
			ID:    "abcdefghijklmnopqrst",
			Names: []string{"/z_test_container"},
		},
		{
			ID:    "1234567890_abcxyzdef",
			Names: []string{"/a_test_container"},
		},
	}

	proxy := new(mockedProxy)
	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	state := &types.ContainerState{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}
	containerJSON := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{State: state}, Config: &container.Config{Tty: false, Image: "alpine"},
		NetworkSettings: &types.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{},
		},
	}

	expected := "INFO Testing inspect image...\n"
	b := make([]byte, 8)

	binary.BigEndian.PutUint32(b[4:], uint32(len(expected)))
	b = append(b, []byte(expected)...)

	reader := io.NopCloser(bytes.NewReader(b))

	createResponse := container.CreateResponse{
		ID: "abcdefghijkl",
	}

	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(containerJSON, nil)
	proxy.On("ContainerStart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerRestart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerStop", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerRemove", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerCreate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(createResponse, nil)
	proxy.On("ImagePull", mock.Anything, "alpine", mock.Anything).Return(reader, nil)

	containerItem, err := client.FindContainerByID("abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")
	assert.Equal(t, containerItem.ID, "abcdefghijkl")

	containerItem, err = client.FindContainerByName("z_test_container")
	require.NoError(t, err, "error should not be thrown")
	assert.Equal(t, containerItem.Name, "z_test_container")
	assert.Equal(t, containerItem.ID, "abcdefghijkl")

	actions := myTypes.ContainerActions
	for _, action := range actions {
		webhookItem := &myTypes.Webhook{
			UUID:          "c3413cb2-c1d2-7e8b-a329-8dff7bcfac86",
			ContainerId:   "abcdefghijkl",
			ContainerName: "z_test_container",
			Host:          "localhost",
			Action:        action,
			Created:       time.Time{},
		}

		containerItem, err := client.ContainerActions(webhookItem)
		if err != nil {
			assert.Nil(t, err, "error should not be thrown")
		} else {
			assert.NotNil(t, containerItem, "container should not be nil")
		}
	}

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerActions_error(t *testing.T) {
	containers := []types.Container{
		{
			ID:    "abcdefghijklmnopqrst",
			Names: []string{"/z_test_container"},
		},
		{
			ID:    "1234567890_abcxyzdef",
			Names: []string{"/a_test_container"},
		},
	}

	proxy := new(mockedProxy)
	client := &httpClient{proxy, filters.NewArgs(), &myTypes.Host{ID: "localhost"}, system.Info{}}

	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, errors.New("test"))

	_, err := client.FindContainerByID("random-id")
	require.Error(t, err, "error should be thrown")

	_, err = client.FindContainerByName("random-name")
	require.Error(t, err, "error should be thrown")

	actions := myTypes.ContainerActions
	for _, action := range actions {
		webhookItem := &myTypes.Webhook{
			UUID:          "c3413cb2-c1d2-7e8b-a329-8dff7bcfac86",
			ContainerId:   "abcdefghijkl",
			ContainerName: "z_test_container",
			Host:          "localhost",
			Action:        action,
			Created:       time.Time{},
		}

		containerItem, err := client.ContainerActions(webhookItem)
		if err != nil {
			assert.NotNil(t, err, "error should be thrown")
		} else {
			assert.Nil(t, containerItem, "container should be nil")
		}
	}

	proxy.AssertExpectations(t)
}
