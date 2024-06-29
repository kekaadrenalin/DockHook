package docker

import (
	"context"

	"testing"

	"github.com/kekaadrenalin/dockhook/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedClient struct {
	mock.Mock
	types.Client
}

func (m *mockedClient) ListContainers() ([]types.Container, error) {
	args := m.Called()
	return args.Get(0).([]types.Container), args.Error(1)
}

func (m *mockedClient) FindContainerByID(id string) (types.Container, error) {
	args := m.Called(id)
	return args.Get(0).(types.Container), args.Error(1)
}

func (m *mockedClient) Events(ctx context.Context, events chan<- types.ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *mockedClient) Host() *types.Host {
	args := m.Called()
	return args.Get(0).(*types.Host)
}

func TestContainerStore_List(t *testing.T) {

	client := new(mockedClient)
	client.On("ListContainers").Return([]types.Container{
		{
			ID:   "1234",
			Name: "test",
		},
	}, nil)
	client.On("Events", mock.Anything, mock.AnythingOfType("chan<- types.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		<-ctx.Done()
	})
	client.On("Host").Return(&types.Host{
		ID: "localhost",
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	store := types.NewContainerStore(ctx, client)
	containers, _ := store.List()

	assert.Equal(t, containers[0].ID, "1234")
}

func TestContainerStore_die(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers").Return([]types.Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
		},
	}, nil)

	client.On("Events", mock.Anything, mock.AnythingOfType("chan<- types.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- types.ContainerEvent)
			events <- types.ContainerEvent{
				Name:    "die",
				ActorID: "1234",
				Host:    "localhost",
			}
			<-ctx.Done()
		})
	client.On("Host").Return(&types.Host{
		ID: "localhost",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	store := types.NewContainerStore(ctx, client)

	// Wait until we get the event
	events := make(chan types.ContainerEvent)
	store.Subscribe(ctx, events)
	<-events

	containers, _ := store.List()
	assert.Equal(t, containers[0].State, "exited")
}
