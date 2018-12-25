package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	validator "gopkg.in/go-playground/validator.v9"
	yaml "gopkg.in/yaml.v2"

	"github.com/alexeldeib/upbound/pkg/types"
	"github.com/alexeldeib/upbound/pkg/util"
)

var applications []*types.ApplicationMetadata
var validate *validator.Validate // Caches struct info, so single global instance.

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	validate = validator.New()

	http.HandleFunc("/create", create)
	http.HandleFunc("/search", search)

	log.Info("Starting up the server.")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf(err.Error())
	}
}

func create(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Please use a PUT request to create an application.", http.StatusBadRequest)
		return
	}
	// Read in body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body of request", http.StatusInternalServerError)
	}
	// Try to parse the metadata content
	metadata := &types.ApplicationMetadata{}
	err = yaml.Unmarshal(body, metadata)
	if err != nil {
		http.Error(w, "Failed to parse YAML input. This likely indicates malformed request body.", http.StatusBadRequest)
		log.Info("YAML parse error")
		return
	}

	// Validate input
	err = validate.Struct(metadata)
	if err != nil {
		// If we fail to validate, automatically return 400
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to validate input of the following parameters:\n"))

		// Be helpful and tell users what fails in their request
		for _, err := range err.(validator.ValidationErrors) {
			fmt.Fprintf(w, "%s has invalid value %s\n", err.Namespace(), err.Value())
		}
		log.Info("Rejected invalid input.")
		return
	}

	// Check if a conflicting application already exists
	if util.CheckTitle(applications, metadata.Title) {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "An application with title %s already exists, please use a unique title.", metadata.Title)
		return
	}

	w.WriteHeader(http.StatusCreated)
	applications = append(applications, metadata)
	log.WithFields(log.Fields{"name": metadata.Title}).Info("Object added")
	return
}

func search(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Please use a POST request to search for an application.", http.StatusBadRequest)
		return
	}
	// Read in body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body of request", http.StatusInternalServerError)
	}

	// Parse it into a struct, but skip validation
	metadata := &types.ApplicationMetadata{}
	err = yaml.Unmarshal(body, metadata)
	if err != nil {
		http.Error(w, "Failed to parse YAML input. This likely indicates malformed request body.", http.StatusBadRequest)
	}

	log.WithFields(log.Fields{"value": *metadata}).Info("Received a value to check!")
	matches := util.Filter(applications, metadata, util.Compare)
	data, err := yaml.Marshal(matches)
	if err != nil {
		http.Error(w, "Failed to marshal search matches. This is likely a server error.", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}
