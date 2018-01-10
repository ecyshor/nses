package internal

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type JobRunner interface {
	Run()
}

type AwsLambdaRunner struct {
	Props *AwsLambdaTemplateProps
}

func (j AwsLambdaRunner) Run(payload []byte) {
	// Create Lambda service client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	client := lambda.New(sess)
	client.Invoke(&lambda.InvokeInput{
		FunctionName: j.Props.FunctionName,
		Payload:      payload,
	})
}
