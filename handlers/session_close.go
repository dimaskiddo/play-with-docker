package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/dimaskiddo/play-with-docker/provisioner"
	"github.com/dimaskiddo/play-with-docker/storage"
	"github.com/gorilla/mux"
)

func CloseSession(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	session, err := core.SessionGet(sessionId)
	if err == storage.NotFoundError {
		rw.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := core.SessionClose(session); err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	volpath, err := provisioner.AbsUserVolumePath(session)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = os.RemoveAll(volpath)

	ResetCookie(rw, req.Host)
}
