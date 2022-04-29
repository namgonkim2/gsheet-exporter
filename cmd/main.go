package main

import (
	server "github.com/gsheet-exporter/pkg/server"
)

func main() {

	exportServer := server.New(":8080")
	exportServer.Start()
}
