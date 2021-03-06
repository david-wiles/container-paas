package internal

import (
	"flag"
	"github.com/docker/docker/client"
	"os"
	"sync"
	"time"
)

// Global contains all global variables and program configuration settings
type Global struct {
	AppMgr *DefaultAppManager
	Logger *Logger

	Docker *client.Client

	Addr          string
	StopTimeout   time.Duration
	StartTimeout  time.Duration
	DockerNetwork string

	Ingress IngressServer
}

// Parse all arguments. Passed arguments take precedence over environment variables
func FromEnv() (*Global, error) {
	addrPtr := flag.String("addr", ":3000", "Address used to listen for connections")
	stopTimeoutPtr := flag.String("stop-timeout", "15s", "Amount of time to wait for a container to stop")
	containerStartTimeout := flag.String("start-timeout", "15s", "Amount of time to wait for a container to start")
	dockerNetwork := flag.String("network", "app-network", "Name of the docker network the app containers are placed in. "+
		"This will be moved into runner-specific configuration soon.")
	useNginx := flag.Bool("nginx", false, "Indicates whether the program will run behind an nginx proxy")
	logLevel := flag.Int("log", 0, "Log level. 0 indicates all logs, 4 indicates none")

	flag.Parse()

	var (
		addr         string = *addrPtr
		stopTimeout  string = *stopTimeoutPtr
		startTimeout string = *containerStartTimeout
		network      string = *dockerNetwork
		ingress      IngressServer
	)

	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	if addr == "" {
		addr = os.Getenv("ADDR")
	}

	if stopTimeout == "" {
		stopTimeout = os.Getenv("DOCKER_STOP_TIMEOUT")
	}

	if startTimeout == "" {
		startTimeout = os.Getenv("CONTAINER_START_TIMEOUT")
	}

	if network == "" {
		network = os.Getenv("DOCKER_NETWORK")
	}

	dockerStopTimeout, err := time.ParseDuration(stopTimeout)
	if err != nil {
		return nil, err
	}

	startTimeoutDuration, err := time.ParseDuration(startTimeout)
	if err != nil {
		return nil, err
	}

	if *useNginx || os.Getenv("USE_NGINX") == "1" {
		ingress = &NginxPorts{
			NginxAppDir: "/etc/nginx/apps",
			confMu:      &sync.Mutex{},
			ports:       [100]bool{},
			apps:        make(map[string]confPortEntry),
		}
	} else {
		ingress = &NoIngress{}
	}

	return &Global{
		AppMgr: &DefaultAppManager{
			apps:  make(map[string]*App),
			appMu: &sync.Mutex{},
		},
		Logger: &Logger{
			infoLog:  os.Stdout,
			errorLog: os.Stderr,
			level:    LogLevel(*logLevel),
		},
		Docker:        docker,
		Addr:          addr,
		StopTimeout:   dockerStopTimeout,
		StartTimeout:  startTimeoutDuration,
		DockerNetwork: network,
		Ingress:       ingress,
	}, nil
}

var G *Global
