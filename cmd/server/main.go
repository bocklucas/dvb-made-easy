package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/offen/restore-manager/internal/api"
	"github.com/offen/restore-manager/internal/config"
	"github.com/offen/restore-manager/internal/docker"
	"github.com/offen/restore-manager/internal/encrypt"
)

func main() {
	port := flag.Int("port", 7331, "HTTP server port")
	configDir := flag.String("config-dir", "/app/config", "Directory for encrypted config storage")
	stagingDir := flag.String("staging-dir", "/staging", "Directory for temporary backup downloads")
	flag.Parse()

	key, err := encrypt.LoadOrCreateKey(*configDir)
	if err != nil {
		log.Fatalf("failed to initialize encryption key: %v", err)
	}

	manifest, err := config.Load(*configDir, key)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("failed to connect to docker: %v", err)
	}
	defer dockerClient.Close()

	router := api.NewRouter(manifest, key, *configDir, *stagingDir, dockerClient)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("offen-restore-manager listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
