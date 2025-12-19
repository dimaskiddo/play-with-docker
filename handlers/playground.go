package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dimaskiddo/play-with-docker/config"
	"github.com/dimaskiddo/play-with-docker/pwd/types"
)

func NewPlayground(rw http.ResponseWriter, req *http.Request) {
	if !ValidateToken(req) {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	var playground types.Playground

	err := json.NewDecoder(req.Body).Decode(&playground)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, "Error creating playground. Got: %v", err)
		return
	}

	newPlayground, err := core.PlaygroundNew(playground)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, "Error creating playground. Got: %v", err)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(newPlayground)
}

func ListPlaygrounds(rw http.ResponseWriter, req *http.Request) {
	if !ValidateToken(req) {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	playgrounds, err := core.PlaygroundList()
	if err != nil {
		log.Printf("Error listing playgrounds. Got: %v\n", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(playgrounds)
}

type PlaygroundConfigurationResponse struct {
	Id                          string        `json:"id"`
	Domain                      string        `json:"domain"`
	DefaultDinDInstanceImage    string        `json:"default_dind_instance_image"`
	AvailableDinDInstanceImages []string      `json:"available_dind_instance_images"`
	AllowWindowsInstances       bool          `json:"allow_windows_instances"`
	DefaultSessionDuration      time.Duration `json:"default_session_duration"`
	DindVolumeSize              string        `json:"dind_volume_size"`
	L2Subdomain                 string        `json:"l2_subdomain"`
	L2SSHPort                   string        `json:"l2_ssh_port"`
	DefaultInstanceImage        string        `json:"default_instance_image"`
	DefaultLimitCPU             string        `json:"default_limit_cpu"`
	DefaultLimitMemory          string        `json:"default_limit_memory"`
	MaxLimitCPU                 string        `json:"max_limit_cpu"`
	MaxLimitMemory              string        `json:"max_limit_memory"`
	MaxLimitProcess             string        `json:"max_limit_process"`
}

func GetCurrentPlayground(rw http.ResponseWriter, req *http.Request) {
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(PlaygroundConfigurationResponse{
		Id:                          playground.Id,
		Domain:                      playground.Domain,
		DefaultDinDInstanceImage:    playground.DefaultDinDInstanceImage,
		AvailableDinDInstanceImages: playground.AvailableDinDInstanceImages,
		AllowWindowsInstances:       playground.AllowWindowsInstances,
		DefaultSessionDuration:      playground.DefaultSessionDuration,
		DindVolumeSize:              playground.DindVolumeSize,
		L2Subdomain:                 config.L2Subdomain,
		L2SSHPort:                   config.L2SSHPort,
		DefaultInstanceImage:        config.DINDImage,
		DefaultLimitCPU:             fmt.Sprintf("%f", config.DefaultLimitCPU),
		DefaultLimitMemory:          fmt.Sprintf("%d", config.DefaultLimitMemory),
		MaxLimitCPU:                 fmt.Sprintf("%f", config.DefaultMaxLimitCPU),
		MaxLimitMemory:              fmt.Sprintf("%d", config.DefaultMaxLimitMemory),
		MaxLimitProcess:             fmt.Sprintf("%d", config.DefaultMaxLimitProcess),
	})
}

func ValidateToken(req *http.Request) bool {
	_, password, ok := req.BasicAuth()
	if !ok {
		return false
	}

	if password != config.AdminToken {
		return false
	}

	return true
}
