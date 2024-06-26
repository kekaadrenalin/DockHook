package command

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Healthcheck(addr string, base string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	if base == "/" {
		base = ""
	}

	url := fmt.Sprintf("%s%s/command", addr, base)

	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	log.Info("Checking health of " + url)
	resp, err := http.Get(url)

	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("command failed with status code %d", resp.StatusCode)
}
