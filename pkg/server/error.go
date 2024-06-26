package server

import (
	"net/http"
)

func (h *handler) error(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
