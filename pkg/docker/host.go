package docker

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/kekaadrenalin/dockhook/pkg/types"
)

func ParseConnection(connection string) (types.Host, error) {
	parts := strings.Split(connection, "|")
	if len(parts) > 2 {
		return types.Host{}, fmt.Errorf("invalid connection string: %s", connection)
	}

	remoteURL, err := url.Parse(parts[0])
	if err != nil {
		return types.Host{}, err
	}

	name := remoteURL.Hostname()
	if len(parts) == 2 {
		name = parts[1]
	}

	basePath, err := filepath.Abs("./certs")
	if err != nil {
		log.Fatalf("error converting certs path to absolute: %s", err)
	}

	host := remoteURL.Hostname()
	if _, err := os.Stat(filepath.Join(basePath, host)); !os.IsNotExist(err) {
		basePath = filepath.Join(basePath, host)
	} else {
		log.Debugf("Remote host certificate path does not exist %s, falling back to default: %s", filepath.Join(basePath, host), basePath)
	}

	caCertPath := filepath.Join(basePath, "ca.pem")
	certPath := filepath.Join(basePath, "cert.pem")
	keyPath := filepath.Join(basePath, "key.pem")

	hasCerts := true
	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		caCertPath = ""
		hasCerts = false
	}

	return types.Host{
		ID:         strings.ReplaceAll(remoteURL.String(), "/", ""),
		Name:       name,
		URL:        remoteURL,
		CertPath:   certPath,
		CACertPath: caCertPath,
		KeyPath:    keyPath,
		ValidCerts: hasCerts,
	}, nil

}
