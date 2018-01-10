package main

import (
	"database/sql"
	"net/http"

	"github.com/ecyshor/nses/internal"
	"github.com/gorilla/mux"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database/postgres"
	log "github.com/sirupsen/logrus"
)

func main() {
	d, err := sql.Open("postgres", "dbname=nses user=nses password=superpassword host=localhost sslmode=disable")
	handleFailure(err)
	internal.Db = d
	driver, err := postgres.WithInstance(d, &postgres.Config{DatabaseName: "nses"})
	if err != nil {
		log.Fatal("Could not create driver instance", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"nses", driver)
	if err != nil {
		log.Fatal("Could not initialize migrations", err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal("Could not run migrations ", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/templates", internal.TemplateHandler).Methods("POST")
	r.HandleFunc("/templates/{templateId}/jobs", internal.JobHandler).Methods("POST")
	http.Handle("/", r)
	log.Info("Migrated nses, binding and starting.")
	err = http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("Error starting", err)
	}
}

func handleFailure(e error) {

	if e != nil {
		panic(e)
	}
}
