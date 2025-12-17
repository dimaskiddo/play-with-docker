package main

import (
	"log"
	"os"
	"time"

	"github.com/dimaskiddo/play-with-docker/config"
	"github.com/dimaskiddo/play-with-docker/docker"
	"github.com/dimaskiddo/play-with-docker/event"
	"github.com/dimaskiddo/play-with-docker/handlers"
	"github.com/dimaskiddo/play-with-docker/id"
	"github.com/dimaskiddo/play-with-docker/k8s"
	"github.com/dimaskiddo/play-with-docker/provisioner"
	"github.com/dimaskiddo/play-with-docker/pwd"
	"github.com/dimaskiddo/play-with-docker/pwd/types"
	"github.com/dimaskiddo/play-with-docker/scheduler"
	"github.com/dimaskiddo/play-with-docker/scheduler/task"
	"github.com/dimaskiddo/play-with-docker/storage"
)

func main() {
	config.ParseFlags()

	e := initEvent()
	s := initStorage()
	df := initDockerFactory(s)
	kf := initK8sFactory(s)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(df, s), provisioner.NewDinD(id.XIDGenerator{}, df, s))
	sp := provisioner.NewOverlaySessionProvisioner(df)

	core := pwd.NewPWD(df, e, s, sp, ipf)

	tasks := []scheduler.Task{
		task.NewCheckPorts(e, df),
		task.NewCheckSwarmPorts(e, df),
		task.NewCheckSwarmStatus(e, df),
		task.NewCollectStats(e, df, s),
		task.NewCheckK8sClusterStatus(e, kf),
		task.NewCheckK8sClusterExposedPorts(e, kf),
	}
	sch, err := scheduler.NewScheduler(tasks, s, e, core)
	if err != nil {
		log.Fatal("Error initializing the scheduler: ", err)
	}

	sch.Start()

	d, err := time.ParseDuration(config.SessionDuration)
	if err != nil {
		log.Fatalf("Cannot parse duration Got: %v", err)
	}

	dindImage := os.Getenv("DIND_IMAGE")
	if dindImage == "" {
		dindImage = "franela/dind:latest"
	}

	playground := types.Playground{
		Domain:                      config.PlaygroundDomain,
		DefaultDinDInstanceImage:    dindImage,
		AvailableDinDInstanceImages: []string{dindImage},
		AllowWindowsInstances:       config.NoWindows,
		DefaultSessionDuration:      d,
		Extras:                      map[string]interface{}{"LoginRedirect": "http://localhost:3000"},
		Privileged:                  true,
		Tasks:                       []string{".*"},
		DockerClientID:              config.DockerClientID,
		DockerClientSecret:          config.DockerClientSecret,
		GithubClientID:              config.GithubClientID,
		GithubClientSecret:          config.GithubClientSecret,
		GoogleClientID:              config.GoogleClientID,
		GoogleClientSecret:          config.GoogleClientSecret,
		AzureClientID:               config.AzureClientID,
		AzureClientSecret:           config.AzureClientSecret,
		AzureTenantID:               config.AzureTenantID,
	}

	if _, err := core.PlaygroundNew(playground); err != nil {
		log.Fatalf("Cannot create default playground. Got: %v", err)
	}

	handlers.Bootstrap(core, e)
	handlers.Register(nil)
}

func initStorage() storage.StorageApi {
	s, err := storage.NewFileStorage(config.SessionsFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error initializing StorageAPI: ", err)
	}
	return s
}

func initEvent() event.EventApi {
	return event.NewLocalBroker()
}

func initDockerFactory(s storage.StorageApi) docker.FactoryApi {
	return docker.NewLocalCachedFactory(s)
}

func initK8sFactory(s storage.StorageApi) k8s.FactoryApi {
	return k8s.NewLocalCachedFactory(s)
}
