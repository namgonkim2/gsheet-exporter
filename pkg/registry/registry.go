package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	client "github.com/gsheet-exporter/internal/registry"
	"github.com/gsheet-exporter/pkg/logger"
)

type Image struct {
	Name   string        `json:"name,omitempty"`
	Tags   []string      `json:"tags,omitempty"`
	Errors []ErrorsImage `json:"errors,omitempty"`
}

type ErrorsImage struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type Registry struct {
	url string
}

type Catalog struct {
	Repositories []string `json:"repositories,omitempty"`
}

var (
	log = logger.GetInstance()
)

func NewRegistry(registryUrl string) (*Registry, error) {
	if registryUrl == "" {
		err := errors.New("url is empty")
		return nil, err
	}

	return &Registry{
		url: registryUrl,
	}, nil
}

// registry alive check
func (registry *Registry) GetRegistry() error {
	err := client.Ping(registry.url)
	if err != nil {
		log.Error.Printf("%s", err)
		return err
	} else {
		log.Info.Println("Registry Server is Ok")
		return nil
	}
}

// Use the image list parsed from Google Sheet to find if there is an image in the registry.
// 구글시트에서 파싱한 이미지 리스트를 활용해 레지스트리 내 이미지가 있는지 찾는다.
func (registry *Registry) FindCopyImageList(imageList []string) ([]string, []string) {
	copyImageList := []string{}
	findFailImgList := []string{}
	imgJsonStruct := Image{}

	for _, image := range imageList {
		img := strings.Split(image, ":")
		// 이미지에 tag가 안들어갔을 경우 예외 처리
		if len(img) <= 1 {
			log.Error.Printf("Image has not tag: %s\n", image)
			findFailImgList = append(findFailImgList, image)
			continue
		}
		imageName := img[0]
		imageTag := img[1]

		res, err := client.ListTags(registry.url, imageName)
		// registry 서버에 문제가 생겼을 때 반환되는 에러
		if err != nil {
			log.Error.Printf("Cannot Get image tags from Registry Server : %v", err)
			findFailImgList = append(findFailImgList, image)
		} else { // 'tags' 데이터 파싱 후 imageTag와 비교해 있으면 true, 없으면 false
			err = json.Unmarshal([]byte(res), &imgJsonStruct)
			if err != nil {
				log.Error.Printf("Cannot Parse Image Json to Struct : %v", err)
				// 이 에러는 json to struct에 문제가 있을 때 반환되는 에러
				findFailImgList = append(findFailImgList, image)
			} else {
				flags := false
				for _, v := range imgJsonStruct.Tags {
					if v == imageTag {
						flags = true
						break
					}
				}
				// false인 이미지 저장
				if flags == false {
					copyImageList = append(copyImageList, image)
				}
			}
		}
	}

	return copyImageList, findFailImgList
}

// Find delete image list that registry save image but not in sheet image list
func (registry *Registry) FindDeleteImageList(imageList []string) []string {
	deleteImageList := []string{}
	CatalogJsonStruct := Catalog{}
	imgJsonStruct := Image{}

	// get image repositories in registry
	getCatalog, err := client.Catalog(registry.url)
	if err != nil {
		log.Error.Printf("Cannot Get image list from Registry Server: %v", err)
		return nil
	}
	err = json.Unmarshal([]byte(getCatalog), &CatalogJsonStruct)
	if err != nil {
		log.Error.Printf("Cannot Parse Catalog Json to Struct: %s, %v", getCatalog, err)
		return nil
	}
	// find image list used repositories
	i := 1
	for _, repo := range CatalogJsonStruct.Repositories {
		getTags, err := client.ListTags(registry.url, repo)
		if err != nil {
			log.Error.Printf("Cannot Get image tags from Registry Server: %s, %v", repo, err)
			continue
		}
		err = json.Unmarshal([]byte(getTags), &imgJsonStruct)
		if err != nil {
			log.Error.Printf("Cannot Parse Image Json to Struct: %s, %v", getTags, err)
			continue
		}
		// 한번이라도 저장된 적이 없는 이미지 패싱
		if imgJsonStruct.Errors != nil {
			log.Error.Printf("Image: %s, Error Code: %v", repo, imgJsonStruct.Errors)
			imgJsonStruct.Errors = nil
			continue
		}

		for _, tags := range imgJsonStruct.Tags {
			image := fmt.Sprintf("%s:%s", repo, tags)
			log.Info.Printf("[%d] %s", i, image)
			// search delete image
			found := search(imageList, image)
			if !found {
				deleteImageList = append(deleteImageList, image)
			}
			i = i + 1
		}

	}

	return deleteImageList
}

func search(list []string, image string) bool {
	for _, item := range list {
		if item == image {
			return true
		}
	}
	return false
}
