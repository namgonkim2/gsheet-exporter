package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gsheet-exporter/pkg/command"
	"github.com/gsheet-exporter/pkg/gsheet"
	logger "github.com/gsheet-exporter/pkg/logger"
	"github.com/gsheet-exporter/pkg/registry"
	"github.com/gsheet-exporter/pkg/skopeo"
)

type Server struct {
	server *http.Server
}

const (
	YYMMDDhhmmss = "20060102-150405" // 2006-01-02 15:04:05
)

func New(addr string) *Server {
	srv := &Server{
		server: &http.Server{
			Addr:    ":8080",
			Handler: routes(),
		},
	}
	return srv
}

func routes() *http.ServeMux {
	r := http.NewServeMux()
	r.HandleFunc("/health", health)
	r.HandleFunc("/sync", sync)
	r.HandleFunc("/export", export)
	r.HandleFunc("/write", write)
	r.HandleFunc("/push/v1", pushv1)
	return r
}

func (s *Server) Start() {
	logger := logger.GetInstance()

	logger.Info.Printf("Fetch Google Sheet: %s, %s\n", os.Getenv("TARGET_SHEETS"), os.Getenv("TARGET_SHEETS_RANGE"))
	logger.Info.Printf("Copy to %s\n", os.Getenv("REGISTRY_URL"))
	logger.Info.Printf("Upload to %s\n", os.Getenv("SCP_DEST"))
	logger.Info.Println("Listening on 8080...")

	if err := s.server.ListenAndServe(); err != nil {
		panic(err)
	}
}

// google sheet and registry instance create
func createInstance() (*gsheet.Gsheet, *registry.Registry, error) {
	gsheetInstance, err := gsheet.NewGsheet()
	if err != nil {
		return nil, nil, err
	}
	registryInstance, err := registry.NewRegistry()
	if err != nil {
		return nil, nil, err
	}
	return gsheetInstance, registryInstance, nil
}

// health checks
func health(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	logger.Info.Println("[/health] Header: ", req.Header.Get("Content-Type"))

	registryUrl := os.Getenv("REGISTRY_URL")
	_, err := registry.Ping(registryUrl)
	if err != nil {
		fmt.Fprintln(w, "Registry Server Fail")
	} else {
		fmt.Fprintf(w, "Registry Server [%s] 200 OK!\n", registryUrl)
	}
}

// synchronize image registry & google sheets
func sync(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	logger.Info.Println("[/sync] Header: ", req.Header.Get("Content-Type"))

	if req != nil && req.Method == "GET" {
		// if want change the target sheet ranges, find params and update target sheet envs
		ranges, ok := req.URL.Query()["range"]
		if ok && len(ranges[0]) >= 1 {
			logger.Info.Println("Target Sheet Range Update to " + string(ranges[0]))
			os.Setenv("TARGET_SHEETS_RANGE", string(ranges[0]))
		}
		// 1. create instance
		gsheetInstance, registryInstance, err := createInstance()
		if err != nil {
			logger.Error.Println(err)
			return
		}
		skopeo := skopeo.GetInstance()
		// 2. get all google sheet image list
		imageList, exceptImageList, err := gsheetInstance.GetGsheet()
		if err != nil {
			logger.Error.Println(err)
			return
		}
		// 3. copy images into registry if not exists
		copyImageList, err := registryInstance.FindCopyImageList(imageList)
		if err != nil {
			logger.Error.Println(err)
			return
		} else {
			fmt.Fprintln(w, "Copy Image List")
			for idx, copyImage := range copyImageList {
				fmt.Fprintf(w, "[%d] %s\n", idx+1, copyImage)
				err := skopeo.Copy(copyImage)
				if err != nil {
					fmt.Fprintln(w, err)
				}
			}
		}
		// 4. delete images where Export option is false in image list
		fmt.Fprintln(w, "Delete Image List")
		for idx, exceptImage := range exceptImageList {
			fmt.Fprintf(w, "[%d] %s\n", idx+1, exceptImage)
			err := skopeo.Delete(exceptImage)
			if err != nil {
				fmt.Fprintln(w, err)
			}
		}
	}

}

// make tar file and export ftp server
func export(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	logger.Info.Println("[/export] Header: ", req.Header.Get("Content-Type"))

	if req != nil && req.Method == "POST" {
		// 1. create tar.gz name
		now := time.Now().UTC()
		tarName := fmt.Sprintf("%s.tar.gz", now.Format(YYMMDDhhmmss))
		fmt.Fprintf(w, "Archiving %s ...\n", tarName)

		// 2. tar command run
		tarCmd := fmt.Sprintf("tar --create --gzip --file=%s -C %s .", tarName, os.Getenv("ARCHIVE_PATH"))
		logger.Info.Println(tarCmd)
		tarOutput, tarErr := command.Run(tarCmd)
		if tarErr != nil {
			logger.Error.Print(tarOutput)
			fmt.Fprintln(w, tarOutput)
			return
		}

		// 3. upload tar file to file repo
		fmt.Fprintf(w, "Uploading %s to %s ...\n", tarName, os.Getenv("SCP_DEST"))
		// sshpass -p{PASSWORD} scp -o StrictHostKeyChecking=no {TAR} {DEST}
		sshCmd := fmt.Sprintf("sshpass -p%s scp -o StrictHostKeyChecking=no %s %s", os.Getenv("SCP_PASS"), tarName, os.Getenv("SCP_DEST"))
		logger.Info.Println(sshCmd)
		sshOutput, sshErr := command.Run(sshCmd)
		if sshErr != nil {
			logger.Error.Print(sshOutput)
			fmt.Fprintln(w, sshOutput)
			return
		}

		// 4. delete tar file
		fmt.Fprintf(w, "Delete %s ...\n", tarName)
		delCmd := fmt.Sprintf("rm -rf %s", tarName)
		logger.Info.Println(delCmd)
		delOutput, delErr := command.Run(delCmd)
		if delErr != nil {
			logger.Error.Print(delOutput)
			fmt.Fprintln(w, delOutput)
		}
	}
}

// write on google sheets what images saved in export files
func write(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	logger.Info.Println("[/write] Header: ", req.Header.Get("Content-Type"))

	// gsheet.SheetWrite(envs)
}

// push v1 based images using docker pull, tag, push
func pushv1(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	logger.Info.Println("[/write] Header: ", req.Header.Get("Content-Type"))

}
