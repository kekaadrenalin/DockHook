package types

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"

	"github.com/puzpuzpuz/xsync/v3"
)

type ContainerStore struct {
	containers              *xsync.MapOf[string, *Container]
	subscribers             *xsync.MapOf[context.Context, chan ContainerEvent]
	newContainerSubscribers *xsync.MapOf[context.Context, chan Container]
	client                  Client
	wg                      sync.WaitGroup
	connected               atomic.Bool
	events                  chan ContainerEvent
	ctx                     context.Context
}

func NewContainerStore(ctx context.Context, client Client) *ContainerStore {
	s := &ContainerStore{
		containers:              xsync.NewMapOf[string, *Container](),
		client:                  client,
		subscribers:             xsync.NewMapOf[context.Context, chan ContainerEvent](),
		newContainerSubscribers: xsync.NewMapOf[context.Context, chan Container](),
		wg:                      sync.WaitGroup{},
		events:                  make(chan ContainerEvent),
		ctx:                     ctx,
	}

	s.wg.Add(1)

	go s.init()

	return s
}

func (s *ContainerStore) checkConnectivity() error {
	if s.connected.CompareAndSwap(false, true) {
		go func() {
			log.Debugf("subscribing to docker events from container store %+v", s.client.Host())
			err := s.client.Events(s.ctx, s.events)
			if !errors.Is(err, context.Canceled) {
				log.Errorf("docker store unexpectedly disconnected from docker events from %+v with %v", s.client.Host(), err)
			}
			s.connected.Store(false)
		}()

		containers, err := s.client.ListContainers()
		if err != nil {
			return err
		}

		s.containers.Clear()
		for _, c := range containers {
			s.containers.Store(c.ID, &c)
		}
	}

	return nil
}

func (s *ContainerStore) List() ([]Container, error) {
	s.wg.Wait()

	if err := s.checkConnectivity(); err != nil {
		return nil, err
	}
	containers := make([]Container, 0)
	s.containers.Range(func(_ string, c *Container) bool {
		containers = append(containers, *c)
		return true
	})

	return containers, nil
}

func (s *ContainerStore) Client() Client {
	return s.client
}

func (s *ContainerStore) Subscribe(ctx context.Context, events chan ContainerEvent) {
	s.subscribers.Store(ctx, events)
}

func (s *ContainerStore) Unsubscribe(ctx context.Context) {
	s.subscribers.Delete(ctx)
}

func (s *ContainerStore) SubscribeNewContainers(ctx context.Context, containers chan Container) {
	s.newContainerSubscribers.Store(ctx, containers)
}

func (s *ContainerStore) init() {
	err := s.checkConnectivity()
	if err != nil {
		panic(any(err))
	}

	s.wg.Done()

	for {
		select {
		case event := <-s.events:
			log.Tracef("received event: %+v", event)
			switch event.Name {
			case "start":
				if container, err := s.client.FindContainerByID(event.ActorID); err == nil {
					log.Debugf("container %s started", container.ID)
					s.containers.Store(container.ID, &container)
					s.newContainerSubscribers.Range(func(c context.Context, containers chan Container) bool {
						select {
						case containers <- container:
						case <-c.Done():
							s.newContainerSubscribers.Delete(c)
						}
						return true
					})
				}
			case "destroy":
				log.Debugf("container %s destroyed", event.ActorID)
				s.containers.Delete(event.ActorID)

			case "die":
				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, bool) {
					if loaded {
						log.Debugf("container %s died", c.ID)
						c.State = "exited"
						return c, false
					}

					return c, true
				})
			case "health_status: healthy", "health_status: unhealthy":
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}

				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, bool) {
					if loaded {
						log.Debugf("health status for container %s is %s", c.ID, healthy)
						c.Health = healthy
						return c, false
					}

					return c, true
				})
			}
			s.subscribers.Range(func(c context.Context, events chan ContainerEvent) bool {
				select {
				case events <- event:
				case <-c.Done():
					s.subscribers.Delete(c)
				}
				return true
			})

		case <-s.ctx.Done():
			return
		}
	}
}
