package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockedProxy struct {
	mock.Mock
	DockerCLI
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

func (m *mockedProxy) TryImagePull(imageName string, registryAuth string) (bool, error) {
	args := m.Called(imageName, registryAuth)

	return args.Get(0).(bool), args.Error(1)
}

func Test_dockerClient_ListContainers_null(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, nil)
	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

	list, err := client.ListContainers()
	assert.Empty(t, list, "list should be empty")
	require.NoError(t, err, "error should not return an error.")

	proxy.AssertExpectations(t)
}

func Test_dockerClient_ListContainers_error(t *testing.T) {
	proxy := new(mockedProxy)
	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

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
	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

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

	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}
	logReader, _ := client.ContainerLogs(context.Background(), id, &since, STDALL)

	actual, _ := io.ReadAll(logReader)
	assert.Equal(t, string(b), string(actual), "message doesn't match expected")
	proxy.AssertExpectations(t)
}

func Test_dockerClient_ContainerLogs_error(t *testing.T) {
	id := "123456"
	proxy := new(mockedProxy)

	proxy.On("ContainerLogs", mock.Anything, id, mock.Anything).Return(nil, errors.New("test"))

	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

	reader, err := client.ContainerLogs(context.Background(), id, nil, STDALL)

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

	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

	containerItem, err := client.FindContainer("abcdefghijkl")
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
	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

	_, err := client.FindContainer("not_valid")
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
	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

	state := &types.ContainerState{Status: "running", StartedAt: time.Now().Format(time.RFC3339Nano)}
	containerJSON := types.ContainerJSON{ContainerJSONBase: &types.ContainerJSONBase{State: state}, Config: &container.Config{Tty: false, Image: "alpine"}}

	expected := "INFO Testing inspect image...\n"
	b := make([]byte, 8)

	binary.BigEndian.PutUint32(b[4:], uint32(len(expected)))
	b = append(b, []byte(expected)...)

	reader := io.NopCloser(bytes.NewReader(b))

	imageJSON := types.ImageInspect{
		ID:          "sha256:abcdefghijkl",
		RepoTags:    nil,
		GraphDriver: types.GraphDriverData{},
		RootFS:      types.RootFS{},
		Metadata:    image.Metadata{},
	}

	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
	proxy.On("ContainerInspect", mock.Anything, "abcdefghijkl").Return(containerJSON, nil)
	proxy.On("ContainerStart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerRestart", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ContainerStop", mock.Anything, "abcdefghijkl", mock.Anything).Return(nil)
	proxy.On("ImageInspectWithRaw", mock.Anything, "alpine").Return(imageJSON, b, nil)
	proxy.On("ImagePull", mock.Anything, "alpine", mock.Anything).Return(reader, nil)

	containerItem, err := client.FindContainer("abcdefghijkl")
	require.NoError(t, err, "error should not be thrown")

	assert.Equal(t, containerItem.ID, "abcdefghijkl")

	actions := ContainerActions
	for _, action := range actions {
		err := client.ContainerActions(action, containerItem.ID, "")
		require.NoError(t, err, "error should not be thrown")
		assert.Equal(t, err, nil)
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
	client := &httpClient{proxy, filters.NewArgs(), &Host{ID: "localhost"}, system.Info{}}

	proxy.On("ContainerList", mock.Anything, mock.Anything).Return(containers, errors.New("test"))
	proxy.On("ContainerStart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))
	proxy.On("ContainerStop", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))
	proxy.On("ContainerRestart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test"))

	containerItem, err := client.FindContainer("random-id")
	require.Error(t, err, "error should be thrown")

	actions := ContainerActions
	for _, action := range actions {
		err := client.ContainerActions(action, containerItem.ID, "")
		require.Error(t, err, "error should be thrown")
		assert.Error(t, err, "error should have been returned")
	}

	proxy.AssertExpectations(t)
}
