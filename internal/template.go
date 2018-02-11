package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type JobType string

const (
	AwsLambda JobType = "awsLambda"
)

type ErrorType int

const (
	Internal ErrorType = iota
	Validation
)

type NsesError struct {
	errorType ErrorType
	message   string
	error     error
}

func (e *NsesError) Error() string {
	return fmt.Sprint(e.message)
}
func (o *JobType) UnmarshalJSON(b []byte) error {
	str := strings.Trim(string(b), `"`)
	switch str {
	case "awsLambda":
		*o = AwsLambda
	default:
		return fmt.Errorf("could not deserialize %s", str)
	}
	return nil
}

type JobTemplate struct {
	Id    *uuid.UUID      `json:"id,omitempty"`
	Type  JobType         `json:"type"`
	Name  *string         `json:"name"`
	Props json.RawMessage `json:"props"`
}

type AwsLambdaTemplateProps struct {
	FunctionName *string `json:"name"`
}

func TemplateHandler(w http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	var jobTemplate = &JobTemplate{}
	err := decoder.Decode(jobTemplate)
	if err != nil {
		log.Error("Invalid JSON", err)
		http.Error(w, "Invalid JSON", 400)
		return
	}
	template, err := CreateTemplate(jobTemplate)
	if err != nil {
		switch errorType := err.(type) {
		case *NsesError:
			switch errorType.errorType {
			case Internal:
				log.Error("could not create template", err)
				http.Error(w, err.Error(), 500)
				return
			case Validation:
				log.Error("could not create template", err)
				http.Error(w, err.Error(), 400)
				return
			}
		default:
			log.Error("could not create template", err)
			http.Error(w, err.Error(), 500)
			return
		}
	}
	templateBytes, err := json.Marshal(template)
	if err != nil {
		log.Error("could not serialize template", err)
		http.Error(w, err.Error(), 500)
	}
	w.Write(templateBytes)
}

func CreateTemplate(jobTemplate *JobTemplate) (*JobTemplate, error) {
	if err := validate(jobTemplate); err != nil {
		return nil, &NsesError{errorType: Validation, error: err}
	}
	res, err := Db.Query("INSERT INTO job_templates(job_type, name,properties) VALUES($1,$2, $3) RETURNING id", jobTemplate.Type, jobTemplate.Name, jobTemplate.Props)
	defer res.Close()
	if err != nil {
		return nil, &NsesError{errorType: Internal, error: err}
	}
	res.Next()
	res.Scan(&jobTemplate.Id)
	return jobTemplate, nil
}

func validate(template *JobTemplate) error {
	switch template.Type {
	case AwsLambda:
		var props AwsLambdaTemplateProps
		json.Unmarshal(template.Props, &props)
		if props.FunctionName == nil {
			return errors.New("lambda ARN is required")

		}
	default:
		return errors.New("invalid template type")
	}
	return nil
}
