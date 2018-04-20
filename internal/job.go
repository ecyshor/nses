package internal

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type Job struct {
	Id          *uuid.UUID     `json:"id"`
	RunInterval *time.Duration `json:"interval"`
	template    *uuid.UUID
	Payload     *json.RawMessage `json:"payload"`
	Path        string           `json:"path"`
}

func JobHandler(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	decoder := json.NewDecoder(request.Body)
	var job = &Job{}
	err := decoder.Decode(job)
	if err != nil {
		log.Error(err)
		http.Error(w, "Invalid JSON", 400)
		return
	}
	template := vars["template"]
	var templateIdStr string
	if err = Db.QueryRow("SELECT id FROM job_templates WHERE name=$1", template).Scan(&templateIdStr); err != nil {
		log.Error("Could not retrieve template id based on name", err)
		http.Error(w, "Could not retrieve id", 500)
		return
	}

	templateId, err := uuid.FromString(templateIdStr)
	if err != nil {
		http.Error(w, "Invalid UUID "+template+" "+templateIdStr, 406)
		return
	}
	defer request.Body.Close()
	job.template = &templateId
	requestPaths := strings.SplitN(request.URL.Path, "/", 5)
	if len(requestPaths) < 5 {
		http.Error(w, "No job identifier provided. please include it in the path after the template.", 400)
		return
	}
	job.Path = strings.Replace(requestPaths[4], "/", ".", -1)
	_, err = Db.Exec("INSERT INTO jobs(interval, template, payload, next_run_min_date, next_run_max_date, path) VALUES($1,$2,$3,$4,$5,$6)",
		job.RunInterval.String(), job.template, job.Payload, time.Now(), time.Now().Add(*job.RunInterval), job.Path)
	if err != nil {
		http.Error(w, "Failed inserting job for template", 500)
		log.Error("Could not insert job.", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
