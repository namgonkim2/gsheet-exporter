package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gsheet-exporter/internal/command"
	"github.com/gsheet-exporter/pkg/gsheet"
	"github.com/gsheet-exporter/pkg/logger"
	"github.com/gsheet-exporter/pkg/registry"
	"github.com/gsheet-exporter/pkg/skopeo"
)

type Server struct {
	server  *http.Server
	handler Handler
}

type Handler struct {
	ServerConfig ServerConfig
}

type ServerConfig struct {
	GoogleConfig   GoogleConfig
	RegistryConfig RegistryConfig
	CredConfig     CredConfig
}

type GoogleConfig struct {
	GoogleCredentials string `required:"true"`
	TargetSheets      string `required:"true"`
	SheetsRange       string `required:"true"`
	ReleaseSheets     string `required:"true"`
}

type RegistryConfig struct {
	RegistryUrl string `required:"true"`
	ArchivePath string `required:"true"`
	ScpDest     string `required:"true"`
	ScpPass     string `required:"true"`
}

type CredConfig struct {
	DockerCred string
	QuayCred   string
	GcrCred    string
}

const (
	YYMMDDhhmmss = "20060102-150405" // 2006-01-02 15:04:05
)

var (
	tarName         string
	imageList       []string
	exceptImageList []string

	log = logger.GetInstance()
)

func New(addr string, srvConfig ServerConfig) *Server {

	h := Handler{
		ServerConfig: srvConfig,
	}

	srv := &Server{
		server: &http.Server{
			Addr: addr,
		},
		handler: h,
	}
	srv.routes()
	return srv
}

func (s *Server) routes() {
	r := http.NewServeMux()
	r.HandleFunc("/", s.handler.controll)
	r.HandleFunc("/health", s.handler.health)
	r.HandleFunc("/sync", s.handler.sync)
	r.HandleFunc("/push/v1", s.handler.pushv1)
	r.HandleFunc("/export", s.handler.export) // export+write

	s.server.Handler = r
}

func (s *Server) Start() {
	log.Info.Printf("Read Google Sheet: %s, %s\n", s.handler.ServerConfig.GoogleConfig.TargetSheets, s.handler.ServerConfig.GoogleConfig.SheetsRange)
	log.Info.Printf("Write Google Sheet: %s, *tar.gz!A1:B\n", s.handler.ServerConfig.GoogleConfig.ReleaseSheets)
	log.Info.Printf("Copy to %s\n", s.handler.ServerConfig.RegistryConfig.RegistryUrl)
	log.Info.Printf("Upload to %s\n", s.handler.ServerConfig.RegistryConfig.ScpDest)
	log.Info.Println("Listening on 8080...")

	if err := s.server.ListenAndServe(); err != nil {
		panic(err)
	}
}

// [api] total task controller : sync & export
func (h *Handler) controll(w http.ResponseWriter, req *http.Request) {
	log.Info.Println("[/] Header: ", req.Header.Get("Content-Type"))
	h.health(w, req)
	h.sync(w, req)
	h.pushv1(w, req)
	h.export(w, req)
}

// [api] registry health check
func (h *Handler) health(w http.ResponseWriter, req *http.Request) {
	log.Info.Println("[/health] Header: ", req.Header.Get("Content-Type"))
	registryInstance, err := registry.NewRegistry(h.ServerConfig.RegistryConfig.RegistryUrl)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	err = registryInstance.GetRegistry()
	if err != nil {
		fmt.Fprintln(w, "Registry Server Fail")
	} else {
		fmt.Fprintf(w, "Registry Server [%s] 200 OK!\n", h.ServerConfig.RegistryConfig.RegistryUrl)
	}
}

// [api] synchronize image registry & google sheets
func (h *Handler) sync(w http.ResponseWriter, req *http.Request) {
	log.Info.Println("[/sync] Header: ", req.Header.Get("Content-Type"))
	imageList = []string{}
	exceptImageList = []string{}

	if req != nil && req.Method == "GET" {
		// if want change the target sheet ranges, find params and update target sheet envs
		ranges, ok := req.URL.Query()["range"]
		if ok && len(ranges[0]) >= 1 {
			log.Info.Println("Target Sheet Range Update to " + string(ranges[0]))
			h.ServerConfig.GoogleConfig.SheetsRange = string(ranges[0])
		}
		// 1. create instance
		gsheetInstance, err := gsheet.NewGsheet(h.ServerConfig.GoogleConfig.GoogleCredentials, h.ServerConfig.GoogleConfig.TargetSheets, h.ServerConfig.GoogleConfig.SheetsRange, "")
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		registryInstance, err := registry.NewRegistry(h.ServerConfig.RegistryConfig.RegistryUrl)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		// 2. get all google sheet image list
		imageList, exceptImageList, err = gsheetInstance.GetGsheet()
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		// 3. copy images into registry if not exists
		copyImageList, failed := registryInstance.FindCopyImageList(imageList)
		fmt.Fprintln(w, "Copy Image List")
		skopeos := skopeo.GetInstance(h.ServerConfig.CredConfig.DockerCred, h.ServerConfig.CredConfig.QuayCred, h.ServerConfig.CredConfig.GcrCred,
			h.ServerConfig.RegistryConfig.RegistryUrl)
		for idx, copyImage := range copyImageList {
			fmt.Fprintf(w, "[%d] %s\n", idx+1, copyImage)
			output, err := skopeos.Copy(copyImage)
			if err != nil {
				fmt.Fprintf(w, "[FAIL][%d] %s:%s", idx+1, copyImage, output)
				failed = append(failed, copyImage)
			}
		}
		if len(failed) > 0 {
			fmt.Fprintln(w, "List of images that failed to find and copy")
			for idx, failImage := range failed {
				fmt.Fprintf(w, "[%d] %s\n", idx+1, failImage)
			}
		}
		// 4. Delete images stored in the registry but not in the Google Sheets list
		fmt.Fprintln(w, "Delete Image List")
		deleteImageList := registryInstance.FindDeleteImageList(imageList)
		for idx, deleteImage := range deleteImageList {
			fmt.Fprintf(w, "[%d] %s\n", idx+1, deleteImage)
			output, err := skopeos.Delete(deleteImage)
			if err != nil {
				fmt.Fprintf(w, "[FAIL][%d] %s:%s", idx+1, deleteImage, output)
			}
		}
	}
}

// [api] push v1 based images using docker pull, tag, push
func (h *Handler) pushv1(w http.ResponseWriter, req *http.Request) {
	log.Info.Println("[/pushv1] Header: ", req.Header.Get("Content-Type"))

	gsheetInstance, err := gsheet.NewGsheet(h.ServerConfig.GoogleConfig.GoogleCredentials, h.ServerConfig.GoogleConfig.TargetSheets, "unsupported!C2:D", "")
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	imageList, exceptImageList, err = gsheetInstance.GetGsheet()
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	for _, image := range imageList {
		output, err := command.DockerCopy(h.ServerConfig.RegistryConfig.RegistryUrl, image)
		if err != nil {
			fmt.Fprintln(w, output)
		} else {
			fmt.Fprintf(w, "Push Docker V1 image : %s\n", image)
		}
	}
}

// [api] make tar file and export ftp server & write on google sheets what images saved in export files
func (h *Handler) export(w http.ResponseWriter, req *http.Request) {
	log.Info.Println("[/export] Header: ", req.Header.Get("Content-Type"))
	tarName = ""

	if req != nil && req.Method == "POST" {
		// 1. create tar.gz name
		now := time.Now()
		tarName = fmt.Sprintf("%s.tar.gz", now.Format(YYMMDDhhmmss))
		fmt.Fprintf(w, "Archiving %s ...\n", tarName)

		// 2. tar command run
		tarCmd := fmt.Sprintf("tar --create --gzip --file=%s -C %s .", tarName, h.ServerConfig.RegistryConfig.ArchivePath)
		log.Info.Println(tarCmd)
		tarOutput, tarErr := command.Run(tarCmd)
		if tarErr != nil {
			log.Error.Print(tarOutput)
			fmt.Fprintln(w, tarOutput)
			return
		}

		// 3. upload tar file to file repo
		fmt.Fprintf(w, "Uploading %s to %s ...\n", tarName, h.ServerConfig.RegistryConfig.ScpDest)
		// sshpass -p{PASSWORD} scp -o StrictHostKeyChecking=no {TAR} {DEST}
		sshCmd := fmt.Sprintf("sshpass -p%s scp -o StrictHostKeyChecking=no %s %s",
			h.ServerConfig.RegistryConfig.ScpPass, tarName, h.ServerConfig.RegistryConfig.ScpDest)
		log.Info.Println(sshCmd)
		sshOutput, sshErr := command.Run(sshCmd)
		if sshErr != nil {
			log.Error.Print(sshOutput)
			fmt.Fprintln(w, sshOutput)
		}

		// 4. delete tar file
		fmt.Fprintf(w, "Delete %s ...\n", tarName)
		delCmd := fmt.Sprintf("rm -rf %s", tarName)
		log.Info.Println(delCmd)
		delOutput, delErr := command.Run(delCmd)
		if delErr != nil {
			log.Error.Print(delOutput)
			fmt.Fprintln(w, delOutput)
		}

		// 5. Create gsheet instance, Add a new sheet and Set write range (target_sheet : {tarName}!B1)
		gsheetInstance, err := gsheet.NewGsheet(h.ServerConfig.GoogleConfig.GoogleCredentials, h.ServerConfig.GoogleConfig.ReleaseSheets, "", "")
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		err = gsheetInstance.AddNewSheet(tarName)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		fmt.Fprintln(w, "Add a new sheet Success")

		// 6. Write image list in new sheet
		gsheetInstance.WriteRange = fmt.Sprintf("%s!A1:B", tarName)
		err = gsheetInstance.SetGsheet(imageList)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		fmt.Fprintln(w, "Write image list in new sheet Success")

	}
}
