package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (h *handler) error(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	log.Debugln("unknown request")
	log.Debugf("RemoteAddr: %s\n", r.RemoteAddr)
	log.Debugf("URL: %s\n", r.URL)
	log.Debugf("Method: %s\n", r.Method)
	log.Debugf("Header: %+v\n", r.Header)

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
