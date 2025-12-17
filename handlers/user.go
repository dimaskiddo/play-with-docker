package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dimaskiddo/play-with-docker/storage"
	"github.com/gorilla/mux"
)

type PublicUserInfo struct {
	Id     string `json:"id"`
	Avatar string `json:"avatar"`
	Name   string `json:"name"`
}

func GetUser(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userId := vars["userId"]

	u, err := core.UserGet(userId)
	if err != nil {
		if storage.NotFound(err) {
			log.Printf("User with id %s was not found\n", userId)
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	pui := PublicUserInfo{Id: u.Id, Avatar: u.Avatar, Name: u.Name}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(pui)
}
