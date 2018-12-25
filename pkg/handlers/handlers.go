package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/alexeldeib/upbound/pkg/types"
	"github.com/alexeldeib/upbound/pkg/util"
	log "github.com/sirupsen/logrus"
	validator "gopkg.in/go-playground/validator.v9"
	yaml "gopkg.in/yaml.v2"
)

// Server represents the global HTTP server and contains global state.
type Server struct {
	Applications []*types.ApplicationMetadata
	Validate     *validator.Validate // Caches struct info, so single global instance.
}

// NewServer prepares a server with handlers, validation, and global application metadata.
func NewServer() Server {
	return Server{Applications: make([]*types.ApplicationMetadata, 0), Validate: validator.New()}
}

// Create handles requests from users to create and persist application metadata.
func (srv *Server) Create(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Failed to parse YAML input. This likely indicates malformed request body. Verify the payload fields and parameter types are correct.", http.StatusBadRequest)
		log.Info("YAML parse error")
		return
	}

	// Validate input
	err = srv.Validate.Struct(metadata)
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
	if util.CheckTitle(srv.Applications, metadata.Title) {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, "An application with title %s already exists, please use a unique title.", metadata.Title)
		return
	}

	w.WriteHeader(http.StatusCreated)
	srv.Applications = append(srv.Applications, metadata)
	log.WithFields(log.Fields{"name": metadata.Title}).Info("Object added")
	return
}

// Search matches user-provided parmaters partially or exactly against existing applications, returning a list of matches.
func (srv *Server) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Please use a POST request to search for an application.", http.StatusBadRequest)
		return
	}
	// Read in body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body of request", http.StatusInternalServerError)
		return
	}

	// Parse it into a struct, but skip validation
	metadata := &types.ApplicationMetadata{}
	err = yaml.Unmarshal(body, metadata)
	if err != nil {
		http.Error(w, "Failed to parse YAML input. This likely indicates malformed request body. Verify the payload fields and parameter types are correct.", http.StatusBadRequest)
		return
	}

	matches := util.Filter(srv.Applications, metadata, util.Compare)
	data, err := yaml.Marshal(matches)
	if err != nil {
		http.Error(w, "Failed to marshal search matches. This is likely a server error.", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}
