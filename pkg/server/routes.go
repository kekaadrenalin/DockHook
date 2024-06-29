package server

import (
	"context"
	"fmt"
	"github.com/kekaadrenalin/dockhook/pkg/types"
	"net/http"
	"path/filepath"
	"strings"

	myErrors "github.com/kekaadrenalin/dockhook/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kekaadrenalin/dockhook/pkg/user"
	"github.com/kekaadrenalin/dockhook/pkg/webhook"
)

type AuthProvider string

const (
	ProviderNone   AuthProvider = "none"
	ProviderSimple AuthProvider = "simple"
	ProviderBasic  AuthProvider = "basic"
)

var ValidAuthProviders = map[string]bool{
	string(ProviderNone):   true,
	string(ProviderSimple): true,
	string(ProviderBasic):  true,
}

// Config is a struct for configuring the web service
type Config struct {
	Base          string
	Addr          string
	Version       string
	Hostname      string
	Authorization Authorization
}

type Authorization struct {
	Provider   AuthProvider
	Authorizer Authorizer
}

type Authorizer interface {
	AuthMiddleware(http.Handler) http.Handler
	CreateToken(string, string) (string, error)
}

type handler struct {
	clients map[string]types.Client
	stores  map[string]*types.ContainerStore
	config  *Config
}

func CreateServer(clients map[string]types.Client, config Config) *http.Server {
	stores := make(map[string]*types.ContainerStore)
	for host, client := range clients {
		stores[host] = types.NewContainerStore(context.Background(), client)
	}

	handler := &handler{
		clients: clients,
		config:  &config,
		stores:  stores,
	}

	return &http.Server{Addr: config.Addr, Handler: createRouter(handler)} //nolint:gosec
}

func createRouter(h *handler) *chi.Mux {
	base := h.config.Base
	r := chi.NewRouter()
	r.Use(cspHeaders)

	if h.config.Authorization.Provider != ProviderNone && h.config.Authorization.Authorizer == nil {
		log.Panic("Authorization provider is set but no authorizer is provided")
	}

	r.Route(base, func(r chi.Router) {
		if h.config.Authorization.Provider != ProviderNone {
			r.Use(h.config.Authorization.Authorizer.AuthMiddleware)
		}

		r.Group(func(r chi.Router) {
			r.Group(func(r chi.Router) {
				if h.config.Authorization.Provider != ProviderNone {
					r.Use(user.RequireAuthentication)
				}

				r.Post("/api/webhooks/{webhookUUID}", h.containerWebhooks)
				r.Get("/version", h.version)
			})

			defaultHandler := http.StripPrefix(strings.Replace(base+"/", "//", "/", 1), http.HandlerFunc(h.error))
			r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
				defaultHandler.ServeHTTP(w, req)
			})
		})

		// Auth
		if h.config.Authorization.Provider == ProviderSimple {
			r.Post("/api/token", h.createToken)
			r.Delete("/api/token", h.deleteToken)
		}

		// Healthcheck
		r.Get("/healthcheck", h.healthcheck)
	})

	if base != "/" {
		r.Get(base, func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, base+"/", http.StatusMovedPermanently)
		})
	}

	return r
}

func (h *handler) webhookFromRequest(r *http.Request) (*types.Webhook, *myErrors.HTTPError) {
	webhookUUID := chi.URLParam(r, "webhookUUID")

	log.Debugf("webhook UUID: %s", webhookUUID)

	if err := uuid.Validate(webhookUUID); err != nil {
		log.Errorf("wrong UUID: %s", webhookUUID)
		log.Infof("Header.RemoteAddr: %+v\n", r.RemoteAddr)
		log.Infof("Header.Authorization: %+v\n", r.Header)

		return nil, &myErrors.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("error: %s", err),
			Err:        err,
		}
	}

	path, err := filepath.Abs("./data/webhooks.yml")
	if err != nil {
		log.Fatalf("Could not find absolute path to webhooks.yml file: %s", err)
	}

	webhooks, err := webhook.ReadWebhooksFromFile(path)
	if err != nil {
		log.Errorf("unknown error: %s", err)

		return nil, &myErrors.HTTPError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	webhookItem := webhooks.Find(webhookUUID)
	if webhookItem == nil {
		log.Errorf("no webhook found: %s", webhookUUID)

		return nil, &myErrors.HTTPError{
			StatusCode: http.StatusNotFound,
			Err:        err,
		}
	}

	return webhookItem, nil
}
