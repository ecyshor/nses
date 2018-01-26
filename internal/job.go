package internal

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type Job struct {
	Id          *uuid.UUID       `json:"id"`
	RunInterval *time.Duration   `json:"interval"`
	template    *uuid.UUID
	Payload     *json.RawMessage `json:"payload"`
}

func JobHandler(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	decoder := json.NewDecoder(request.Body)
	var job Job
	err := decoder.Decode(&job)
	if err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	templateId, err := uuid.FromString(vars["templateId"])
	if err != nil {
		http.Error(w, "Invalid UUID", 406)
		return
	}
	job.template = &templateId
	_, err = Db.Query("INSERT INTO jobs(interval,template, payload, next_run_min_date, next_run_max_date) VALUES($1,$2,$3,$4,$5)", job.RunInterval.String(), job.template, job.Payload, time.Now(), time.Now().Add(*job.RunInterval))
	if err != nil {
		http.Error(w, "Failed inserting job for template", 500)
		log.Error("Could not insert job.", err)
		return
	}
	defer request.Body.Close()
}
