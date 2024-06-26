package command

import (
	"main/pkg/user"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	argsType "main/pkg/types"
)

func CreateUser(args argsType.Args) (user.User, error) {
	if args.CreateUserCmd.Username == "" || args.CreateUserCmd.Password == "" {
		log.Fatal("Username and password are required")
	}

	path, err := filepath.Abs("./data/users.yml")
	if err != nil {
		log.Fatalf("Could not find absolute path to users.yml file: %s", err)
	}

	return user.CreateUser(path, user.User{
		Username: args.CreateUserCmd.Username,
		Password: args.CreateUserCmd.Password,
		Name:     args.CreateUserCmd.Name,
		Email:    args.CreateUserCmd.Email,
	}, true)
}
