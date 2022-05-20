package command

import (
	"fmt"
	"os/exec"

	"github.com/gsheet-exporter/pkg/logger"
)

const (
	DOCKER_PULL = "docker pull %s"      // image
	DOCKER_TAG  = "docker tag %s %s/%s" // image url/image
	DOCKER_PUSH = "docker push %s/%s"
)

var (
	log = logger.GetInstance()
)

func Run(cmdString string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdString)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func DockerCopy(registryUrl, image string) (string, error) {
	pull := fmt.Sprintf(DOCKER_PULL, image)
	log.Info.Println(pull)
	output, err := Run(pull)
	if err != nil {
		log.Error.Printf("Cannot docker pull : %s", output)
		return output, err
	}
	tag := fmt.Sprintf(DOCKER_TAG, image, registryUrl, image)
	log.Info.Println(tag)
	output, err = Run(tag)
	if err != nil {
		log.Error.Printf("Cannot docker tag : %s", output)
		return output, err
	}
	push := fmt.Sprintf(DOCKER_PUSH, registryUrl, image)
	log.Info.Println(push)
	output, err = Run(push)
	if err != nil {
		log.Error.Printf("Cannot docker push : %s", output)
		return output, err
	}
	return output, nil
}
