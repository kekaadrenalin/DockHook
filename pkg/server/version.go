package server

import (
	"fmt"
	"net/http"
)

func (h *handler) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	_, _ = fmt.Fprintf(w, "<pre>%v</pre>", h.config.Version)
}
