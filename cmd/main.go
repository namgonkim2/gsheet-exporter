package main

import (
	"os"

	"github.com/gsheet-exporter/pkg/logger"
	server "github.com/gsheet-exporter/pkg/server"
)

func main() {

	// find blank environments
	checkEnvrionments()

	exportServer := server.New(":8080")
	exportServer.Start()
}

func checkEnvrionments() {
	logger := logger.GetInstance()

	envs := map[string]string{
		"GOOGLE_APPLICATION_CREDENTIALS": os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		"TARGET_SHEETS":                  os.Getenv("TARGET_SHEETS"),
		"TARGET_SHEETS_RANGE":            os.Getenv("TARGET_SHEETS_RANGE"),
		"REGISTRY_URL":                   os.Getenv("REGISTRY_URL"),
		"ARCHIVE_PATH":                   os.Getenv("ARCHIVE_PATH"),
		"SCP_DEST":                       os.Getenv("SCP_DEST"),
		"SCP_PASS":                       os.Getenv("SCP_PASS"),
		"DOCKER_CRED":                    os.Getenv("DOCKER_CRED"),
		"QUAY_CRED":                      os.Getenv("QUAY_CRED"),
		"GCR_CRED":                       os.Getenv("GCR_CRED"),
	}

	for key, env := range envs {
		if key == "DOCKER_CRED" || key == "QUAY_CRED" || key == "GCR_CRED" {
			continue
		}

		if env == "" {
			logger.Error.Printf("No specified necessary envs '%s'", key)
			panic(env)
		}
	}

}
