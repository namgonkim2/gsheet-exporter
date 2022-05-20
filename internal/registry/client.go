package registry

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gsheet-exporter/pkg/logger"
)

var (
	log = logger.GetInstance()
)

// Health check registry server
func Ping(url string) error {
	srv := fmt.Sprintf("http://%s/v2", url)
	resp, err := http.Get(srv)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		_, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

// image list(no have image tags)
func Catalog(url string) (string, error) {
	srv := fmt.Sprintf("http://%s/v2/_catalog", url)
	resp, err := http.Get(srv)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		bodyString := string(bodyBytes)
		return bodyString, nil
	}
	return "", nil
}

// image list tags
func ListTags(url string, image string) (string, error) {
	srv := fmt.Sprintf("http://%s/v2/%s/tags/list", url, image)
	resp, err := http.Get(srv)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyString := string(bodyBytes)
	return bodyString, nil

}
