package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ecyshor/nses/internal"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	"github.com/mattes/migrate"
	"github.com/mattes/migrate/database/postgres"
	_ "github.com/mattes/migrate/source/file"
	log "github.com/sirupsen/logrus"
	"github.com/golang/protobuf/proto"
)

func main() {
	log.SetLevel(log.DebugLevel)
	/*if status, err := strconv.ParseBool(getEnv("DEBUG", "false")); err != nil && status {
	}*/
	nsesDb := getEnv("NSES_DB_DB", "nses")
	d, err := sql.Open("postgres", fmt.Sprintf("dbname=%s user=%s password=%s host=%s sslmode=disable",
		nsesDb, getEnv("NSES_DB_USER", "nses"), getEnv("NSES_DB_PASSWORD", "superpassword"),
		getEnv("NSES_DB_HOST", "localhost")))
	handleFailure(err)
	internal.Db = d
	driver, err := postgres.WithInstance(d, &postgres.Config{DatabaseName: nsesDb})
	if err != nil {
		log.Panic("Could not create driver instance", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		nsesDb, driver)
	if err != nil {
		log.Fatal("Could not initialize migrations", err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatal("Could not run migrations ", err)
	}
	r := mux.NewRouter()
	r.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	r.HandleFunc("/templates", internal.TemplateHandler).Methods("PUT")
	r.PathPrefix("/templates/{template}/jobs").Handler(http.HandlerFunc(internal.JobHandler)).Methods("POST")
	http.Handle("/", r)
	log.Info("Migrated nses, binding and starting.")
	go internal.Start()
	srv := &http.Server{
		Handler:      r,
		Addr:         ":8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

// GRPC SERVER
type GrpcNsesServer struct {
}

var marshaller = &jsonpb.Marshaler{EnumsAsInts: true}

func (s *GrpcNsesServer) CreateTemplate(ctx context.Context, template *JobTemplate) (*JobTemplate, error) {
	var message proto.Message
	var jobType internal.JobType

	if template.GetLambdaProperties() != nil {
		message = template.GetLambdaProperties()
		jobType = internal.AwsLambda
	}
	if template.GetLambdaProperties() != nil {
		message = template.GetHttpProperties()
		jobType = internal.Http
	}
	value, e := marshaller.MarshalToString(message)
	if e != nil {
		return nil, e
	}
	jobTemplate, err := internal.CreateTemplate(&internal.JobTemplate{Type: jobType, Props: []byte(value)})
	if err != nil {
		log.Error("could not create template", err)
		return nil, err
	}
	template.Id = jobTemplate.Id.String()
	return template, nil
}

func (s *GrpcNsesServer) CreateJob(context.Context, *Job) (*Job, error) {
	return nil, nil
}

func handleFailure(e error) {

	if e != nil {
		panic(e)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
