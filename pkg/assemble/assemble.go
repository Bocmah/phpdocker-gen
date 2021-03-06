package assemble

import (
	"fmt"
	"strings"

	"github.com/Bocmah/phpdocker-gen/internal/dockercompose"
	"github.com/Bocmah/phpdocker-gen/pkg/service"
)

// DockerCompose assembles dockercompose config from service.FullConfig
func DockerCompose(conf *service.FullConfig) *dockercompose.Config {
	compose := &dockercompose.Config{
		Version: "3.8",
	}

	appName := formatAppName(conf.AppName)

	if conf.Services.PresentServicesCount() > 1 {
		compose.Networks = dockercompose.Networks{createDefaultNetwork(appName)}
	}

	if conf.Services.IsPresent(service.Database) {
		compose.Volumes = dockercompose.NamedVolumes{createDefaultVolume(appName)}
	}

	optsAssembler := &optionsAssembler{
		compose:      compose,
		serviceFiles: conf.GetServiceFiles(),
		serviceEnv:   conf.GetEnvironment(),
	}

	if conf.Services.IsPresent(service.Database) {
		optsAssembler.databaseSystemInUse = conf.Services.Database.System
	}

	for _, s := range service.SupportedServices() {
		if !conf.Services.IsPresent(s) {
			continue
		}

		assembler := NewServiceAssembler(s)

		compose.Services = append(compose.Services, assembler(conf, optsAssembler.assembleForService(s)...))
	}

	return compose
}

func formatAppName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

func createDefaultNetwork(appName string) *dockercompose.Network {
	return &dockercompose.Network{
		Name:   fmt.Sprintf("%s-network", appName),
		Driver: dockercompose.NetworkDriverBridge,
	}
}

func createDefaultVolume(appName string) *dockercompose.NamedVolume {
	return &dockercompose.NamedVolume{
		Name:   fmt.Sprintf("%s-data", appName),
		Driver: dockercompose.VolumeDriverLocal,
	}
}
