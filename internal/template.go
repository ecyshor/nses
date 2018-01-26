package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type JobType string

const (
	AwsLambda JobType = "awsLambda"
)

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
	Id    *string         `json:"id,omitempty"`
	Type  *JobType        `json:"type"`
	Props json.RawMessage `json:"props"`
}

type AwsLambdaTemplateProps struct {
	FunctionName *string `json:"name"`
}

func TemplateHandler(w http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	var jobTemplate JobTemplate
	err := decoder.Decode(&jobTemplate)
	if err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	templateType := *jobTemplate.Type
	switch templateType {
	case AwsLambda:
		var props AwsLambdaTemplateProps
		json.Unmarshal(jobTemplate.Props, &props)
		if props.FunctionName == nil {
			http.Error(w, "Lambda ARN is required", 400)
			return
		}
	default:
		http.Error(w, "Invalid template type", 400)
		return
	}
	_, err = Db.Query("INSERT INTO job_templates(job_type, properties) VALUES($1,$2)", jobTemplate.Type, jobTemplate.Props)
	if err != nil {
		http.Error(w, "Failed inserting template", 500)
		return
	}
	defer request.Body.Close()
}
