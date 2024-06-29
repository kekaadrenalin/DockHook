package docker

import (
	"context"

	"testing"

	myTypes "github.com/kekaadrenalin/dockhook/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedClient struct {
	mock.Mock
	myTypes.Client
}

func (m *mockedClient) ListContainers() ([]myTypes.Container, error) {
	args := m.Called()
	return args.Get(0).([]myTypes.Container), args.Error(1)
}

func (m *mockedClient) FindContainerByID(id string) (myTypes.Container, error) {
	args := m.Called(id)
	return args.Get(0).(myTypes.Container), args.Error(1)
}

func (m *mockedClient) Events(ctx context.Context, events chan<- myTypes.ContainerEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *mockedClient) Host() *myTypes.Host {
	args := m.Called()
	return args.Get(0).(*myTypes.Host)
}

func TestContainerStore_List(t *testing.T) {

	client := new(mockedClient)
	client.On("ListContainers").Return([]myTypes.Container{
		{
			ID:   "1234",
			Name: "test",
		},
	}, nil)
	client.On("Events", mock.Anything, mock.AnythingOfType("chan<- docker.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		<-ctx.Done()
	})
	client.On("Host").Return(&myTypes.Host{
		ID: "localhost",
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	store := myTypes.NewContainerStore(ctx, client)
	containers, _ := store.List()

	assert.Equal(t, containers[0].ID, "1234")
}

func TestContainerStore_die(t *testing.T) {
	client := new(mockedClient)
	client.On("ListContainers").Return([]myTypes.Container{
		{
			ID:    "1234",
			Name:  "test",
			State: "running",
		},
	}, nil)

	client.On("Events", mock.Anything, mock.AnythingOfType("chan<- types.ContainerEvent")).Return(nil).
		Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)
			events := args.Get(1).(chan<- myTypes.ContainerEvent)
			events <- myTypes.ContainerEvent{
				Name:    "die",
				ActorID: "1234",
				Host:    "localhost",
			}
			<-ctx.Done()
		})
	client.On("Host").Return(&myTypes.Host{
		ID: "localhost",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	store := myTypes.NewContainerStore(ctx, client)

	// Wait until we get the event
	events := make(chan myTypes.ContainerEvent)
	store.Subscribe(ctx, events)
	<-events

	containers, _ := store.List()
	assert.Equal(t, containers[0].State, "exited")
}
