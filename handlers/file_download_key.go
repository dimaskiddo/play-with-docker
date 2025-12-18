package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func fileDownloadKey(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s, _ := core.SessionGet(sessionId)
	if s == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	i := core.InstanceGet(s, instanceName)
	if i == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	instanceFile, err := core.InstanceFile(i, "/root/.ssh/id_rsa")
	if err != nil {
		log.Println("Error getting ssh private key file:", err)
		rw.WriteHeader(http.StatusNotFound)

		return
	}

	rw.Header().Set("Content-Type", "application/octet-stream")
	rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"keyfile_%s_%s.pem\"", sessionId, i.Hostname))
	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	rw.Header().Set("Pragma", "no-cache")
	rw.Header().Set("Expires", "0")

	if _, err = io.Copy(rw, instanceFile); err != nil {
		log.Println("Error writing ssh key to response:", err)
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}
}
