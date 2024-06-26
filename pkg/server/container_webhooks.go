package server

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (h *handler) containerWebhooks(w http.ResponseWriter, r *http.Request) {
	webhookItem, myErr := h.webhookFromRequest(r)
	if myErr != nil {
		w.WriteHeader(myErr.StatusCode)

		if myErr.Message != "" {
			_, _ = fmt.Fprintf(w, myErr.Message)
		}

		return
	}

	client, ok := h.clients[webhookItem.Host]
	if !ok {
		log.Errorf("no client found for host %v", webhookItem.Host)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	container, err := client.FindContainer(webhookItem.ContainerId)
	if err != nil {
		log.Errorf("no container found %s", webhookItem.ContainerId)

		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = client.ContainerActions(webhookItem.Action, container.ID)
	if err != nil {
		log.Errorf("error while trying to perform action %s: %s", webhookItem.Action, err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof("container action performed: %s; container id: %s", webhookItem.Action, container.ID)

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "OK")
}
