package server

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/kekaadrenalin/dockhook/pkg/types"
)

func (h *handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Trace("Executing command request")
	var client types.Client

	for _, v := range h.clients {
		client = v
		break
	}

	if ping, err := client.Ping(r.Context()); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		_, _ = fmt.Fprintf(w, "OK API Version %v", ping.APIVersion)
	}
}
