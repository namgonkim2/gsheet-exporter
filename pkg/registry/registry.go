package registry

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	client "github.com/gsheet-exporter/internal/registry"
	"github.com/gsheet-exporter/pkg/logger"
)

type Image struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type Registry struct {
	url string
}

func NewRegistry() (*Registry, error) {

	if os.Getenv("REGISTRY_URL") == "" {
		err := errors.New("url is empty")
		return nil, err
	}

	return &Registry{
		url: os.Getenv("REGISTRY_URL"),
	}, nil
}

func Ping(url string) (bool, error) {
	logger := logger.GetInstance()
	resp, err := client.Ping(url)
	if err != nil {
		logger.Error.Printf("%s", err)
	} else {
		logger.Info.Println("Registry Server is Ok")
	}
	return resp, err
}

// 시트 리스트 이미지를 하나씩 레지스트리 내 있는지 검사
func (registry *Registry) FindCopyImageList(imageList []string) ([]string, error) {
	result := []string{}
	data := Image{}

	for _, image := range imageList {

		img := strings.Split(image, ":")
		imageName := img[0]
		imageTag := img[1]

		res, err := client.ListTags(registry.url, imageName)
		if err != nil {
			return []string{}, err
		} else {
			// 'tags' 데이터 파싱 후 imageTag와 비교해 있으면 exist, 없으면 no exist
			json.Unmarshal([]byte(res), &data)
			flags := false
			for _, v := range data.Tags {
				if v == imageTag {
					flags = true
					break
				}
			}
			if flags == false {
				result = append(result, image)
			}
		}
	}

	return result, nil
}
