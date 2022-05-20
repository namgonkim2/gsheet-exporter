package main

import (
	"flag"

	"github.com/gsheet-exporter/pkg/logger"
	"github.com/gsheet-exporter/pkg/server"
)

var (
	log = logger.GetInstance()
)

func main() {

	// check environments
	envs := checkEnvFlags()

	exportServer := server.New(":8080", server.ServerConfig{
		GoogleConfig: server.GoogleConfig{
			GoogleCredentials: *envs["GOOGLE_APPLICATION_CREDENTIALS"],
			TargetSheets:      *envs["TARGET_SHEETS"],
			SheetsRange:       *envs["SHEETS_RANGE"],
			ReleaseSheets:     *envs["RELEASE_SHEETS"],
		},
		RegistryConfig: server.RegistryConfig{
			RegistryUrl: *envs["REGISTRY_URL"],
			ArchivePath: *envs["ARCHIVE_PATH"],
			ScpDest:     *envs["SCP_DEST"],
			ScpPass:     *envs["SCP_PASS"],
		},
		CredConfig: server.CredConfig{
			DockerCred: *envs["DOCKER_CRED"],
			QuayCred:   *envs["QUAY_CRED"],
			GcrCred:    *envs["GCR_CRED"],
		},
	})
	exportServer.Start()
}

func checkEnvFlags() map[string]*string {
	// flag declare (using camelcase)
	envs := map[string]*string{
		"GOOGLE_APPLICATION_CREDENTIALS": flag.String("googleAppCreds", "./credentials.json", "[string] google creds key file path"),
		"TARGET_SHEETS":                  flag.String("targetSheets", "", "[string] read to target sheets"),
		"SHEETS_RANGE":                   flag.String("sheetsRange", "CK1!C2:D,CK2!C2:D", "[string] target google sheets cell ranges"),
		"RELEASE_SHEETS":                 flag.String("releaseSheets", "", "[string] write on release sheets"),
		"REGISTRY_URL":                   flag.String("registryUrl", "", "[string] private registry url"),
		"ARCHIVE_PATH":                   flag.String("archivePath", "", "[string] where you saved tar.gz file"),
		"SCP_DEST":                       flag.String("scpDest", "", "[string] scp destination"),
		"SCP_PASS":                       flag.String("scpPass", "", "[string] scp passwd"),
		"DOCKER_CRED":                    flag.String("dockerCred", "", "[string] docker credentials"),
		"QUAY_CRED":                      flag.String("quayCred", "", "[string] quay cred"),
		"GCR_CRED":                       flag.String("gcrCred", "", "[string] gcr cred"),
	}
	flag.Parse()

	// check requires...
	for key, env := range envs {
		if key == "DOCKER_CRED" || key == "QUAY_CRED" || key == "GCR_CRED" {
			continue
		}

		if *env == "" {
			log.Error.Printf("No specified necessary envs '%s'", key)
			panic(env)
		}
	}
	return envs
}
