package internal

import (
	"database/sql"
	"encoding/json"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/lambda"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"io/ioutil"
)

var Db *sql.DB

type JobExecutor interface {
}
type Jobber struct {
	jobs chan RunnableJob
}

type JobResult struct {
	job   *Job
	error error
	extra []byte
}

//TODO think about finding way to make it easier to run with multiple instances
func Start() {
	ticker := time.NewTicker(30 * time.Second)
	jobs := make(chan RunnableJob, 100)
	doneJobs := make(chan JobResult, 100)
	jobber := NewRunner(jobs)
	log.Info("Starting scheduler.")
	go jobber.Start(doneJobs)
	go StartMarker(doneJobs)
	runJobs(jobber)
	defer close(jobs)
	defer ticker.Stop()
	for range ticker.C {
		runJobs(jobber)
	}

}

func StartMarker(marking chan JobResult) {
	log.Info("Starting job marker")
	for runJob := range marking {
		runJob.MarkRun()
	}
	log.Info("Finished job marker")
}

func runJobs(jobber *Jobber) {
	//TODO stream the jobs from postgresql
	//TODO ensure jobs are not picked up twice, keep in memory status
	jobsToRun, err := retrieveForDate(time.Now().Add(2 * time.Hour))
	log.Infof("Running %d jobs", len(jobsToRun))
	if err != nil {
		log.Error("Could not retrieve jobs to run", err)
	}
	for _, job := range jobsToRun {
		template, err := job.withTemplate()
		if err != nil {
			log.Error("Could not retrieve job template", err)
		}
		jobber.jobs <- RunnableJob{
			&job, template,
		}
	}
}

func NewRunner(jobs chan RunnableJob) *Jobber {
	return &Jobber{jobs}
}

func (j Jobber) Start(marking chan JobResult) {
	log.Info("Starting job runner")
	for job := range j.jobs {
		log.Debug("Running job")
		marking <- job.Run()
	}
	log.Info("Finished job runner.")
}

type RunnableJob struct {
	job      *Job
	template *JobTemplate
}

type GenericErrorMessage struct {
	Message string `json:"message"`
}

func (j RunnableJob) Run() JobResult {
	switch j.template.Type {
	case AwsLambda:
		{
			// Create Lambda service client
			var props AwsLambdaTemplateProps
			json.Unmarshal(j.template.Props, &props)
			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))
			client := lambda.New(sess)
			output, err := client.Invoke(&lambda.InvokeInput{
				FunctionName: props.FunctionName,
				Payload:      *j.job.Payload,
			})
			if err != nil {
				log.Error("Exception while invoking lambda job", err)
				errorMessage, _ := json.Marshal(GenericErrorMessage{err.Error()})
				return JobResult{j.job, err, errorMessage}
			}
			if math.Mod(float64(*output.StatusCode), 10) > 2 {
				bytes, e := json.Marshal(output)
				if e != nil {
					log.Error("Could not serialize output to json for lambda function", e)
				}
				return JobResult{j.job, err, bytes}
			}
		}
	case Http:
		{
			var props HttpTemplateProps
			var jobVariables map[string]string
			json.Unmarshal(j.template.Props, &props)
			json.Unmarshal(*j.job.Payload, &jobVariables)
			var url = *props.url
			for key := range jobVariables {
				url = strings.Replace(jobVariables[key], ":"+key, jobVariables[key], -1)
			}
			resp, err := http.NewRequest(*props.method, url, nil)
			if err != nil {
				log.Error("Could not serialize output to json for http integration", err)
				return JobResult{j.job, err, nil}
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Error("Could not read responde body", err)
			}
			defer resp.Body.Close()
			return JobResult{j.job, err, body}
		}
	}
	return JobResult{j.job, nil, make([]byte, 0)}
}

func (r *JobResult) MarkRun() {
	j := r.job
	log.Info("Marking job run.")
	rows, err := Db.Query("SELECT run_date FROM job_runs WHERE job_id = $1 ORDER BY run_date DESC LIMIT 5", j.Id)
	defer rows.Close()
	if err != nil {
		log.Error("Could not retrieve last run dates", err)
		panic(err)
	}
	lastRunDates := make([]time.Time, 5, 5)
	for rows.Next() {
		var runTime time.Time
		if err := rows.Scan(&runTime); err != nil {
			log.Error("Error reading run date", err)
		}
		lastRunDates = append(lastRunDates, runTime)
	}
	var nextMin, nextMax time.Time
	jobRunDate := time.Now()
	if len(lastRunDates) == 5 {
		fixedNext := lastRunDates[4].Add(5 * *j.RunInterval)
		nextMin = fixedNext.Add(time.Duration(int64(float64(-0.2) * float64(j.RunInterval.Nanoseconds()))))
		nextMax = fixedNext.Add(time.Duration(int64(float64(0.2) * float64(j.RunInterval.Nanoseconds()))))
	} else {
		nextMin = jobRunDate.Add(time.Duration(int64(float64(0.9) * float64(j.RunInterval.Nanoseconds()))))
		nextMax = jobRunDate.Add(time.Duration(int64(float64(1.1) * float64(j.RunInterval.Nanoseconds()))))
	}
	log.Infof("Calculated next times: min [%s], max [%s]", nextMin.String(), nextMax.String())
	_, err = Db.Exec("UPDATE jobs SET next_run_max_date = $1, next_run_min_date = $2 WHERE id = $3", nextMax, nextMin, j.Id)
	if err != nil {
		log.Error("Error updating jobs dates", err)
	}
	var succesfull int
	if r.error != nil {
		succesfull = 0
	} else {
		succesfull = 1
	}
	_, err = Db.Exec("INSERT INTO job_runs(job_id, run_date,successfull, extra_details) VALUES ($1,$2, $3, $4)", j.Id,
		jobRunDate, succesfull, r.extra)
	if err != nil {
		log.Error("Error inserting job run", err)
	}
	log.Info("Marked job run.")

}

func retrieveForDate(toDate time.Time) ([]Job, error) {
	rows, err := Db.Query("SELECT id, template ,payload, INTERVAL  FROM jobs WHERE (next_run_min_date >= $1 AND next_run_max_date <= $2) OR next_run_max_date <= $1", time.Now(), toDate)
	defer rows.Close()
	if err != nil {
		log.Error("Could not retrieve mandatory jobs to run", err)
		return nil, err
	}
	var jobs []Job
	for rows.Next() {
		var job Job
		var duration string
		if err := rows.Scan(&job.Id, &job.template, &job.Payload, &duration); err != nil {
			log.Error("Could not scan row to job", err)
			return nil, err
		}
		ival, err := time.ParseDuration(duration)
		if err != nil {
			log.Error("Could not parse duration for job", err)
			return nil, err
		}
		job.RunInterval = &ival
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (j *Job) withTemplate() (*JobTemplate, error) {
	row := Db.QueryRow("SELECT job_type,properties FROM job_templates WHERE id = $1", j.template)
	var template JobTemplate
	if err := row.Scan(&template.Type, &template.Props); err != nil {
		log.Error("Could not retrieve template for job", err, j)
		return nil, err
	}
	return &template, nil
}
