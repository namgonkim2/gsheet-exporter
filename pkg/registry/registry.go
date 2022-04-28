package exporter

import (
	"encoding/json"
	"fmt"
	"strings"

	client "github.com/gsheet-exporter/internal/registry"
)

type Image struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func Ping(url string) (bool, error) {
	resp, err := client.Ping(url)
	return resp, err
}

// 시트 리스트 이미지를 하나씩 레지스트리 내 있는지 검사
func Sync(sheetList []string, url string) ([]string, error) {
	result := []string{}
	data := Image{}

	for _, image := range sheetList {

		img := strings.Split(image, ":")
		imageName := img[0]
		imageTag := img[1]

		res, err := client.ListTags(url, imageName)
		if err != nil {
			fmt.Println(err)
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
			if flags == true {
				result = append(result, fmt.Sprintf("%s => already exist", image))
			} else {
				result = append(result, fmt.Sprintf("%s => no exist", image))
			}
		}
	}

	return result, nil
}
