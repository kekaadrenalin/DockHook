package types

var (
	Version = "head"
)

type Args struct {
	Addr                 string              `arg:"env:DOCKHOOK_ADDR" default:":8080" help:"sets host:port to bind for server. This is rarely needed inside a docker container."`
	Base                 string              `arg:"env:DOCKHOOK_BASE" default:"/" help:"sets the base for http router."`
	Hostname             string              `arg:"env:DOCKHOOK_HOSTNAME" help:"sets the hostname for display. This is useful with multiple DockHook instances."`
	AuthProvider         string              `arg:"--auth-provider,env:DOCKHOOK_AUTH_PROVIDER" default:"basic" help:"sets the auth provider to use. Currently only simple and basic is supported."`
	Level                string              `arg:"env:DOCKHOOK_LEVEL" default:"info" help:"set DockHook log level. Use debug for more logging."`
	WaitForDockerSeconds int                 `arg:"--wait-for-docker-seconds,env:DOCKHOOK_WAIT_FOR_DOCKER_SECONDS" help:"wait for docker to be available for at most this many seconds before starting the server."`
	FilterStrings        []string            `arg:"env:DOCKHOOK_FILTER,--filter,separate" help:"filters docker containers using Docker syntax."`
	Filter               map[string][]string `arg:"-"`
	RemoteHost           []string            `arg:"env:DOCKHOOK_REMOTE_HOST,--remote-host,separate" help:"list of hosts to connect remotely"`

	HealthcheckCmd   *HealthcheckCmd   `arg:"subcommand:command" help:"checks if the server is running"`
	CreateUserCmd    *CreateUserCmd    `arg:"subcommand:create-user" help:"creates a new user and saves it in configuration file for simple auth"`
	CreateWebhookCmd *CreateWebhookCmd `arg:"subcommand:create-webhook" help:"creates a new webhook and saves it in configuration file"`
}

type HealthcheckCmd struct {
}

type CreateUserCmd struct {
	Username    string `arg:"positional"`
	Password    string `arg:"--password, -p" help:"sets the password for the user"`
	Name        string `arg:"--name, -n" help:"sets the display name for the user"`
	Email       string `arg:"--email, -e" help:"sets the email for the user"`
	WithoutSave bool   `arg:"--without-save, -w" help:"don't save the user to file"`
}

type CreateWebhookCmd struct {
	DockerComposeOnly bool `arg:"--docker-compose-only, -o" help:"find only docker compose container'"`
}

func (Args) Version() string {
	return Version
}
