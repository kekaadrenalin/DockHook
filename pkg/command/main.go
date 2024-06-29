package command

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/kekaadrenalin/dockhook/pkg/docker"
	"github.com/kekaadrenalin/dockhook/pkg/server"
	"github.com/kekaadrenalin/dockhook/pkg/types"
	"github.com/kekaadrenalin/dockhook/pkg/user"
)

func Default(args types.Args) {
	if !server.ValidAuthProviders[args.AuthProvider] {
		log.Fatalf("Invalid auth provider %s", args.AuthProvider)
	}

	log.Infof("DockHook version %s", types.Version)

	clients := docker.CreateClients(args)

	srv := createServer(args, clients)
	go func() {
		log.Infof("Accepting connections on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	stop()

	log.Info("shutting down gracefully, press Ctrl+C again to force")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	defer cancel()

	log.Debug("shutdown complete")
}

func createServer(args types.Args, clients map[string]types.Client) *http.Server {
	var provider = server.ProviderNone
	var authorizer server.Authorizer

	if args.AuthProvider != string(server.ProviderNone) {
		path, err := filepath.Abs("./data/users.yml")
		if err != nil {
			log.Fatalf("Could not find absolute path to users.yml file: %s", err)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatalf("Could not find users.yml file at %s", path)
		}

		users, err := user.ReadUsersFromFile(path)
		if err != nil {
			log.Fatalf("Could not read users.yml file at %s: %s", path, err)
		}

		if args.AuthProvider == string(server.ProviderSimple) {
			provider = server.ProviderSimple
			authorizer = user.NewSimpleAuth(users)
		} else if args.AuthProvider == string(server.ProviderBasic) {
			provider = server.ProviderBasic
			authorizer = user.NewBasicAuth(users)
		}
	}

	config := server.Config{
		Addr:     args.Addr,
		Base:     args.Base,
		Version:  types.Version,
		Hostname: args.Hostname,
		Authorization: server.Authorization{
			Provider:   provider,
			Authorizer: authorizer,
		},
	}

	return server.CreateServer(clients, config)
}
