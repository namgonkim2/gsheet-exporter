package skopeo

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gsheet-exporter/internal/command"
	"github.com/gsheet-exporter/pkg/logger"
)

type Skopeo struct {
	DockerCred string
	QuayCred   string
	GCrGred    string

	CopyTo string
}

const (
	CHECK      = "skopeo inspect docker://%s"                                                    // image
	CRED_CHECK = "skopeo inspect --creds=%s docker://%s"                                         // cred , image
	COPY       = "skopeo copy --dest-tls-verify=false docker://%s docker://%s/%s"                // src_image, dest, dest_image
	CRED_COPY  = "skopeo copy --src-creds=%s --dest-tls-verify=false docker://%s docker://%s/%s" // src_cred, src_image, dest, dest_image
	DELETE     = "skopeo delete --tls-verify=false docker://%s/%s"                               // url/image
)

var (
	skopeo *Skopeo
	once   sync.Once

	log = logger.GetInstance()
)

func GetInstance(dockerCred, quayCred, gcrCred, copyTo string) *Skopeo {
	once.Do(func() {
		skopeo = New(dockerCred, quayCred, gcrCred, copyTo)
	})
	SetProfiles(skopeo)

	return skopeo
}

func New(dockerCred, quayCred, gcrCred, copyTo string) *Skopeo {
	return &Skopeo{
		DockerCred: dockerCred,
		QuayCred:   quayCred,
		GCrGred:    gcrCred,
		CopyTo:     copyTo,
	}
}

// set creds about image repository & regax repo string
func SetProfiles(skopeo *Skopeo) {
	/* py3 - 필요한 이유 조사가 필요함
	    def __init__(self, docker_cred, quay_cred, gcr_cred, copy_to):
			self.profiles = {
				'docker.io': {'pattern': re.compile('^[a-z0-9.]*docker.io/'), 'cred': docker_cred},
				'docker.elastic.co': {'pattern': re.compile('^docker.elastic.co/'), 'cred': ''},
				'public.ecr.aws': {'pattern': re.compile('^public.ecr.aws/'), 'cred': ''},
				'ghcr.io': {'pattern': re.compile('^[a-z0-9.]*ghcr.io/'), 'cred': ''},
				'quay.io': {'pattern': re.compile('^[a-z0-9.]*quay.io/'), 'cred': quay_cred},
				'gcr': {'pattern': re.compile('^[a-z0-9.]*gcr.io/'), 'cred': gcr_cred}
			}
			self.copy_to = copy_to
	*/

}

func (skopeo *Skopeo) Inspect(image string) error {
	var cmd string
	if skopeo.DockerCred == "" {
		cmd = fmt.Sprintf(CHECK, image)
	} else {
		cmd = fmt.Sprintf(CRED_CHECK, skopeo.DockerCred, image)
	}
	log.Info.Println(cmd)
	output, err := command.Run(cmd)
	if err != nil {
		log.Error.Print(output)
		return err
	}
	return nil
}

func (skopeo *Skopeo) Copy(image string) (string, error) {
	var cmd string
	if skopeo.DockerCred == "" {
		cmd = fmt.Sprintf(COPY, image, skopeo.CopyTo, image)
	} else {
		cmd = fmt.Sprintf(CRED_COPY, skopeo.DockerCred, image, skopeo.CopyTo, image)
	}
	log.Info.Println(cmd)
	output, err := command.Run(cmd)
	if err != nil {
		log.Error.Print(output)
		return output, err
	}
	return output, nil
}

func (skopeo *Skopeo) Delete(image string) (string, error) {
	cmd := fmt.Sprintf(DELETE, skopeo.CopyTo, image)
	log.Info.Println(cmd)
	output, err := command.Run(cmd)
	if err != nil {
		if strings.Contains(output, "Image may not exist or is not stored with a v2 Schema in a v2 registry") == true {
			log.Info.Printf("[%s] Not Exists in Registry", image)
			return output, nil
		} else {
			log.Error.Print(output)
			return output, err
		}
	}
	return output, nil
}
