package ratelimit

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"

	"fmt"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

func NewEC2Metadata(sess *session.Session) *EC2Metadata {
	// Create a EC2Metadata client from just a session.
	return &EC2Metadata{Client: *ec2metadata.New(sess)}
}

type EC2Metadata struct {
	Client ec2metadata.EC2Metadata
}

func (r *EC2Metadata) GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return r.Client.GetInstanceIdentityDocument()
}

func NewAutoScaling(sess *session.Session) *AutoScaling {
	// Create a Session with a custom region
	return &AutoScaling{Client: autoscaling.New(sess)}
}

type AutoScaling struct {
	Client autoscalingiface.AutoScalingAPI
}

func (c *AutoScaling) DescribeAutoScalingGroups(input *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return c.Client.DescribeAutoScalingGroups(input)
}

func NewEC2(sess *session.Session) *EC2 {
	return &EC2{Client: ec2.New(sess)}
}

type EC2 struct {
	Client ec2iface.EC2API
}

func (c *EC2) DescribeTags(input *ec2.DescribeTagsInput) (*ec2.DescribeTagsOutput, error) {
	return c.Client.DescribeTags(input)
}

func (c *EC2) GetAsgName(instanceID string) (string, error) {
	filters := ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("resource-id"), Values: []*string{aws.String(instanceID)}},
			{Name: aws.String("key"), Values: []*string{aws.String("aws:autoscaling:groupName")}},
		},
	}

	tags, err := c.DescribeTags(&filters)
	if err != nil {
		return "", err
	}

	fmt.Println("Tags: ", len(tags.Tags))
	if len(tags.Tags) == 0 {
		return "", fmt.Errorf("Instance tags empty")
	}

	//fmt.Println("yeap, ", tags.Tags[0], " final")
	return *tags.Tags[0].Value, nil
}
