package main

import (
	"os"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
	commands "main/pkg/command"
	argsType "main/pkg/types"

	"github.com/alexflint/go-arg"
)

func main() {
	args, subcommand := parseArgs()
	validateEnvVars()

	if subcommand != nil {
		switch subcommand.(type) {
		case *argsType.HealthcheckCmd:
			if err := commands.Healthcheck(args.Addr, args.Base); err != nil {
				log.Fatal(err)
			}

		case *argsType.CreateUserCmd:
			newUser, err := commands.CreateUser(args)
			if err != nil {
				log.Fatalf("Could not create new user: %s", err)
			}

			log.Infof("User %s successfully saved", newUser.Username)
			log.Infof("Password hash: %s", newUser.Password)

		case *argsType.CreateWebhookCmd:
			webhook, err := commands.CreateWebhook(args)
			if err != nil {
				log.Fatalf("Could not create new webhook: %s", err)
			}

			log.Infoln("Webhook successfully saved", webhook)
			log.Infof("UUID: %s", webhook.UUID)
		}

		os.Exit(0)
	}

	commands.Default(args)
}

func parseArgs() (argsType.Args, interface{}) {
	var args argsType.Args
	parser := arg.MustParse(&args)

	configureLogger(args.Level)

	args.Filter = make(map[string][]string)

	for _, filter := range args.FilterStrings {
		pos := strings.Index(filter, "=")
		if pos == -1 {
			parser.Fail("each filter should be of the form key=value")
		}

		key := filter[:pos]
		val := filter[pos+1:]
		args.Filter[key] = append(args.Filter[key], val)
	}

	return args, parser.Subcommand()
}

func configureLogger(level string) {
	if l, err := log.ParseLevel(level); err == nil {
		log.SetLevel(l)
	} else {
		panic(any(err))
	}

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})

}

func validateEnvVars() {
	argsType := reflect.TypeOf(argsType.Args{})
	expectedEnvs := make(map[string]bool)

	for i := 0; i < argsType.NumField(); i++ {
		field := argsType.Field(i)

		for _, tag := range strings.Split(field.Tag.Get("arg"), ",") {
			if strings.HasPrefix(tag, "env:") {
				expectedEnvs[strings.TrimPrefix(tag, "env:")] = true
			}
		}
	}

	for _, env := range os.Environ() {
		actual := strings.Split(env, "=")[0]

		if strings.HasPrefix(actual, "DOCKHOOK_") && !expectedEnvs[actual] {
			log.Warnf("Unexpected environment variable %s", actual)
		}
	}
}
