package internal

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

type TimeUnit int

const (
	Millisecond TimeUnit = iota
	Second
	Minute
	Hour
	Day
	Week
	Month
	Year
)

type Interval struct {
	unit  TimeUnit
	value int
}

type Job struct {
	id          *uuid.UUID
	runInterval Interval
	weight      int
	template    uuid.UUID
	payload     json.RawMessage
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
		http.Error(w, "Invalid template id", 400)
		return
	}
	job.template = templateId
	_, err = Db.Query("INSERT INTO jobs(interval,weight,template, payload) VALUES($1,$2,$3,$4)", job.runInterval, job.weight, job.template, job.payload)
	if err != nil {
		http.Error(w, "Failed inserting job for template", 500)
		return
	}
	defer request.Body.Close()
}
