package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gsheet-exporter/pkg/gsheet"
	logger "github.com/gsheet-exporter/pkg/logger"
	"github.com/gsheet-exporter/pkg/registry"
)

type Server struct {
	server *http.Server
}

var (
	envs map[string]string
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
	r.HandleFunc("/", index)
	r.HandleFunc("/health", health)
	r.HandleFunc("/sync", sync)
	r.HandleFunc("/write", write)
	return r
}

func (s *Server) Start() {

	logger := logger.GetInstance()

	envs = map[string]string{
		"GOOGLE_APPLICATION_CREDENTIALS": os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		"TARGET_SHEETS":                  os.Getenv("TARGET_SHEETS"),
		"TARGET_SHEETS_RANGE":            os.Getenv("TARGET_SHEETS_RANGE"),
		"REGISTRY_URL":                   os.Getenv("REGISTRY_URL"),
		"ARCHIVE_PATH":                   os.Getenv("ARCHIVE_PATH"),
		"SCP_DEST":                       os.Getenv("SCP_DEST"),
		"SCP_PASS":                       os.Getenv("SCP_PASS"),
	}

	for key, env := range envs {
		if env == "" {
			logger.Error.Printf("No specified necessary envs '%s'", key)
			panic(env)
		}
	}

	gsheet.NewGsheet(envs)

	logger.Info.Printf("Fetch Google Sheet: %s, %s\n", envs["TARGET_SHEETS"], envs["TARGET_SHEETS_RANGE"])
	logger.Info.Printf("Copy to %s\n", envs["REGISTRY_URL"])
	logger.Info.Printf("Upload to %s\n", envs["SCP_DEST"])
	logger.Info.Println("Listening on 8080...")

	if err := s.server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	logger := logger.GetInstance()
	logger.Info.Println("{}")
}

func health(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	if req != nil {
		if req.Method == "GET" {
			logger.Info.Println("GET /health")
		} else if req.Method == "POST" {
			logger.Info.Println("POST /health")
		}
		logger.Info.Println("Header: ", req.Header.Get("Content-Type"))
	}
	for key, env := range envs {
		if key == "SCP_PASS" {
			continue
		}
		envString := fmt.Sprintf("%s: %s", key, env)
		fmt.Fprintln(w, envString)
	}
	_, err := registry.Ping(envs["REGISTRY_URL"])
	if err != nil {
		logger.Error.Printf("%s", err)
		fmt.Println(w, "Registry Server Fail!")
	} else {
		logger.Info.Println("Registry Server is fine")
		fmt.Fprintf(w, "Registry Server [%s] 200 OK!\n", envs["REGISTRY_URL"])
	}
}

func sync(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	if req != nil {
		logger.Info.Printf("%s /sync\n", req.Method)
		logger.Info.Println("Header: ", req.Header.Get("Content-Type"))
		// get으로 받았을 때 gsheet와 registry의 이미지 리스트를 sync할 예정
		if req.Method == "GET" {
			logger.Info.Println("Read Google Sheet")
			sheetList, err := gsheet.SheetRead(envs)
			if err != nil {
				logger.Error.Printf("%s", err)
			}
			logger.Info.Println("Sync with Private Registry...")
			result, err := registry.Sync(sheetList, envs["REGISTRY_URL"])
			if err != nil {
				logger.Error.Printf("%s", err)
			} else {
				logger.Info.Println("OK!")
				for _, v := range result {
					logger.Info.Println(v)
				}
			}
		}
	}

}

func write(w http.ResponseWriter, req *http.Request) {
	logger := logger.GetInstance()
	if req != nil {
		logger.Info.Println("GET /write\nHeader: ", strings.Split(req.Header.Get("Accept"), ";"))
	}
	logger.Info.Println(w, "write on Google Sheet")
	gsheet.SheetWrite(envs)
}
