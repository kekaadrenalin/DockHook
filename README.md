# DockHook

DockHook allows you to control your Docker containers through a webhook system.

[![License](https://img.shields.io/github/license/kekaadrenalin/dockhook)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/kekaadrenalin/dockhook)](https://goreportcard.com/report/github.com/kekaadrenalin/dockhook)
[![Docker Pulls](https://img.shields.io/docker/pulls/kekaadrenalin/dockhook.svg)](https://hub.docker.com/r/kekaadrenalin/dockhook/)
[![Docker Version](https://img.shields.io/docker/v/kekaadrenalin/dockhook?sort=semver)](https://hub.docker.com/r/kekaadrenalin/dockhook/)
![Test](https://github.com/kekaadrenalin/DockHook/workflows/Test/badge.svg)

This project was inspired by the excellent projects [Dozzle](https://github.com/amir20/dozzle) (which allows you to view
container logs in real-time and provides basic functionality for starting and stopping containers)
and [Portainer](https://www.portainer.io/) (a project for deploying and managing complex Docker environments). If you
find the current functionality insufficient, you may consider using these projects.

## Getting Started

Pull the latest release with:

    $ docker pull kekaadrenalin/dockhook:latest

### Running DockHook

The simplest way to use DockHook is to run the docker container. Also, mount the Docker Unix socket with `--volume`
to `/var/run/docker.sock`:

    $ docker run --name dockhook -d --volume=/var/run/docker.sock:/var/run/docker.sock:ro -p 8888:8080 kekaadrenalin/dockhook:latest

DockHook will be available at [http://localhost:8888/](http://localhost:8888/).

Here is the Docker Compose file:

    version: "3"
    services:
      dockhook:
        container_name: dockhook
        image: kekaadrenalin/dockhook:latest
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock:ro
          - ./data/users.yml:/data/users.yml:rw
          - ./data/webhooks.yml:/data/webhooks.yml:rw
          ... or ...
          - ./some_data/:/data/
        ports:
          - 8888:8080

### Authorization

You need to run a command at least once to add a new user to the file storage (which must be accessible to the
container, for example, `./data/users.yml`). The file storage will be created in any case in the volume:

    $ docker run kekaadrenalin/dockhook create-user admin --password password --email test@email.net --name "John Doe"

### Webhooks

Additionally, you need to create the first webhook interactively to manage the desired container. The available actions
are `START`, `STOP`, `RESTART`, and `PULL` (more details can be found in the Actions section):

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock:ro kekaadrenalin/dockhook create-webhook

You can also quickly filter only the containers started via `docker compose`:

    $ docker run --volume=/var/run/docker.sock:/var/run/docker.sock:ro kekaadrenalin/dockhook create-webhook --docker-compose-only

### Actions

List of available actions:

- `START`: starts an existing stopped container
- `STOP`: stops an existing running container
- `RESTART`: restarts an existing running container
- `PULL`: pulls and updates the latest version of the image and restarts the existing running container

## License

DockHook is distributed under [AGPL-3.0-only](LICENSE).
