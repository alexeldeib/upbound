package main

import (
	"net/http"
	"os"

	"github.com/alexeldeib/upbound/pkg/handlers"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	server := handlers.NewServer()

	http.HandleFunc("/create", server.Create)
	http.HandleFunc("/search", server.Search)

	log.Info("Starting up the server.")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf(err.Error())
	}
}
