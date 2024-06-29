package docker

import (
	"context"
	"errors"
	"testing"

	myTypes "github.com/kekaadrenalin/dockhook/pkg/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type fakeCLI struct {
	myTypes.DockerCLI
	mock.Mock
}

func (f *fakeCLI) ContainerList(context.Context, container.ListOptions) ([]types.Container, error) {
	args := f.Called()
	return args.Get(0).([]types.Container), args.Error(1)
}

func (f *fakeCLI) Info(context.Context) (system.Info, error) {
	return system.Info{}, nil
}

func Test_valid_localhost(t *testing.T) {
	client := new(fakeCLI)
	client.On("ContainerList").Return([]types.Container{}, nil)
	fakeClientFactory := func(_ map[string][]string) (myTypes.Client, error) {
		return NewClient(client, filters.NewArgs(), &myTypes.Host{
			ID: "localhost",
		}), nil
	}

	args := myTypes.Args{}

	actualClient, _ := createLocalClient(args, fakeClientFactory)

	assert.NotNil(t, actualClient)
	client.AssertExpectations(t)
}

func Test_invalid_localhost(t *testing.T) {
	client := new(fakeCLI)
	client.On("ContainerList").Return([]types.Container{}, errors.New("error"))
	fakeClientFactory := func(_ map[string][]string) (myTypes.Client, error) {
		return NewClient(client, filters.NewArgs(), &myTypes.Host{
			ID: "localhost",
		}), nil
	}

	args := myTypes.Args{}

	actualClient, _ := createLocalClient(args, fakeClientFactory)

	assert.Nil(t, actualClient)
	client.AssertExpectations(t)
}

func Test_valid_remote(t *testing.T) {
	local := new(fakeCLI)
	local.On("ContainerList").Return([]types.Container{}, errors.New("error"))
	fakeLocalClientFactory := func(_ map[string][]string) (myTypes.Client, error) {
		return NewClient(local, filters.NewArgs(), &myTypes.Host{
			ID: "localhost",
		}), nil
	}

	remote := new(fakeCLI)
	remote.On("ContainerList").Return([]types.Container{}, nil)
	fakeRemoteClientFactory := func(_ map[string][]string, host myTypes.Host) (myTypes.Client, error) {
		return NewClient(remote, filters.NewArgs(), &myTypes.Host{
			ID: "test",
		}), nil
	}

	args := myTypes.Args{
		RemoteHost: []string{"tcp://test:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory, "")

	assert.Equal(t, 1, len(clients))
	assert.Contains(t, clients, "test")
	assert.NotContains(t, clients, "localhost")
	local.AssertExpectations(t)
	remote.AssertExpectations(t)
}

func Test_valid_remote_and_local(t *testing.T) {
	local := new(fakeCLI)
	local.On("ContainerList").Return([]types.Container{}, nil)
	fakeLocalClientFactory := func(_ map[string][]string) (myTypes.Client, error) {
		return NewClient(local, filters.NewArgs(), &myTypes.Host{
			ID: "localhost",
		}), nil
	}

	remote := new(fakeCLI)
	remote.On("ContainerList").Return([]types.Container{}, nil)
	fakeRemoteClientFactory := func(_ map[string][]string, host myTypes.Host) (myTypes.Client, error) {
		return NewClient(remote, filters.NewArgs(), &myTypes.Host{
			ID: "test",
		}), nil
	}
	args := myTypes.Args{
		RemoteHost: []string{"tcp://test:2375"},
	}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory, "")

	assert.Equal(t, 2, len(clients))
	assert.Contains(t, clients, "test")
	assert.Contains(t, clients, "localhost")
	local.AssertExpectations(t)
	remote.AssertExpectations(t)
}

func Test_no_clients(t *testing.T) {
	local := new(fakeCLI)
	local.On("ContainerList").Return([]types.Container{}, errors.New("error"))
	fakeLocalClientFactory := func(_ map[string][]string) (myTypes.Client, error) {

		return NewClient(local, filters.NewArgs(), &myTypes.Host{
			ID: "localhost",
		}), nil
	}
	fakeRemoteClientFactory := func(_ map[string][]string, host myTypes.Host) (myTypes.Client, error) {
		client := new(fakeCLI)
		return NewClient(client, filters.NewArgs(), &myTypes.Host{
			ID: "test",
		}), nil
	}

	args := myTypes.Args{}

	clients := createClients(args, fakeLocalClientFactory, fakeRemoteClientFactory, "")

	assert.Equal(t, 0, len(clients))
	local.AssertExpectations(t)
}
