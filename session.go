package ratelimit

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func NewAwsSessionWithRegion(region string) *session.Session {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	return sess
}

func NewAwsSession() *session.Session {
	sess := session.Must(session.NewSession())
	return sess
}
