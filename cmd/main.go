package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	gsheet "github.com/gsheet-exporter/pkg/gsheet"
	registry "github.com/gsheet-exporter/pkg/registry"
)

func main() {

	envs := map[string]string{
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
			log.Fatalf("[FATAL]: no specified necessary envs %s", key)
		}
	}

	fmt.Printf("Fetch Google Sheet: %s, %s\n", envs["TARGET_SHEETS"], envs["TARGET_SHEETS_RANGE"])
	fmt.Printf("Copy to %s\n", envs["REGISTRY_URL"])
	fmt.Printf("Upload to %s\n", envs["SCP_DEST"])
	fmt.Println("Listening on 8080...")

	// Check the Server on, Registry on
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		if req != nil {
			fmt.Println("GET /health\nHeader: ", strings.Split(req.Header.Get("Accept"), ";"))
		}
		fmt.Fprintf(w, "Server is Running!\n\n")
		for key, env := range envs {
			envString := fmt.Sprintf("%s: %s", key, env)
			fmt.Fprintln(w, envString)
		}
		fmt.Fprintf(w, "Registry Server Check...")
		resp, err := registry.Ping(envs["REGISTRY_URL"])
		if resp != true {
			fmt.Fprintf(w, "Fail!\n%s", err)
		} else {
			fmt.Fprintln(w, "200 OK!")
		}

	})

	// Fetch google sheet, Check the Registry image is exist
	http.HandleFunc("/sync", func(w http.ResponseWriter, req *http.Request) {
		if req != nil {
			fmt.Println("GET /sync\nHeader: ", strings.Split(req.Header.Get("Accept"), ";"))
		}
		fmt.Fprintln(w, "Sync 'Google Sheet' and 'Private Registry'")
		fmt.Fprintln(w, "Read Google Sheet...")

		sheetList, err := gsheet.SheetRead(envs)
		if err != nil {
			fmt.Fprintf(w, "Fail!\n%s", err)
		}
		fmt.Fprintf(w, "Sync with Private Registry...")
		result, err := registry.Sync(sheetList, envs["REGISTRY_URL"])
		if err != nil {
			fmt.Fprintf(w, "Fail!\n%s", err)
		} else {
			fmt.Fprintln(w, "OK!")
			for _, v := range result {
				fmt.Fprintln(w, v)
			}
		}

	})

	http.HandleFunc("/write", func(w http.ResponseWriter, req *http.Request) {
		if req != nil {
			fmt.Println("GET /write\nHeader: ", strings.Split(req.Header.Get("Accept"), ";"))
		}
		fmt.Fprintln(w, "write on Google Sheet")
		gsheet.SheetWrite(envs)
	})

	http.ListenAndServe(":8080", nil)
}
