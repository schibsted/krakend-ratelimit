//
// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
//
package ratelimit

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

type MockedAutoscaling struct {
	autoscalingiface.AutoScalingAPI
	Resp *autoscaling.DescribeAutoScalingGroupsOutput
}

func (c MockedAutoscaling) DescribeAutoScalingGroups(input *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	if c.Resp != nil {
		return c.Resp, nil
	}

	return c.Resp, fmt.Errorf("Error")
}

type EC2Mock struct {
	ec2iface.EC2API
	Resp *ec2.DescribeTagsOutput
}

func (c EC2Mock) DescribeTags(input *ec2.DescribeTagsInput) (*ec2.DescribeTagsOutput, error) {
	if c.Resp != nil {
		return c.Resp, nil
	}

	return c.Resp, fmt.Errorf("Error")
}

// A EC2Metadata is an EC2 Metadata service Client.
type EC2MetadataMock struct {
	ok bool
}

func (e EC2MetadataMock) GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	if e.ok {
		return ec2metadata.EC2InstanceIdentityDocument{}, nil
	}

	return ec2metadata.EC2InstanceIdentityDocument{}, fmt.Errorf("Error")
}

func TestAutoScaling_DescribeAutoScalingGroups(t *testing.T) {
	checks := []struct {
		resp        *autoscaling.DescribeAutoScalingGroupsOutput
		expectedErr bool
	}{
		{resp: &autoscaling.DescribeAutoScalingGroupsOutput{}, expectedErr: false},
		{resp: nil, expectedErr: true},
	}

	for _, c := range checks {
		c := c
		t.Run(fmt.Sprintf("Mock: Response: %#v expects %t", c.resp, c.expectedErr), func(t *testing.T) {
			autoScaling := AutoScaling{Client: MockedAutoscaling{Resp: c.resp}}

			got, err := autoScaling.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})

			if c.expectedErr {
				if err == nil {
					t.Errorf("An error is expected")
				}
			} else {
				if err != nil {
					t.Error("An error is not expected", err)
				}

				if got != c.resp {
					t.Errorf("got != expected")
				}
			}
		})
	}
}

func TestEC2_DescribeTags(t *testing.T) {
	checks := []struct {
		resp        *ec2.DescribeTagsOutput
		expectedErr bool
	}{
		{resp: &ec2.DescribeTagsOutput{}, expectedErr: false},
		{resp: nil, expectedErr: true},
		{resp: nil, expectedErr: true},
	}

	for _, c := range checks {
		c := c
		t.Run(fmt.Sprintf("Mock: Response: %#v expects %t", c.resp, c.expectedErr), func(t *testing.T) {
			ec2Client := EC2{Client: EC2Mock{Resp: c.resp}}

			got, err := ec2Client.DescribeTags(&ec2.DescribeTagsInput{})

			if c.expectedErr {
				if err == nil {
					t.Errorf("An error is expected")
				}
			} else {
				if err != nil {
					t.Error("An error is not expected", err)
				}

				if got != c.resp {
					t.Errorf("got != expected")
				}
			}
		})
	}
}

func TestEC2_GetAsgName(t *testing.T) {

	asgName := "TestAsg"
	tags := []*ec2.TagDescription{{Value: aws.String(asgName)}}
	emptyTags := []*ec2.TagDescription{}

	checks := []struct {
		instanceID  string
		resp        *ec2.DescribeTagsOutput
		expectedErr bool
	}{
		{instanceID: "", resp: &ec2.DescribeTagsOutput{Tags: tags}, expectedErr: false},
		{instanceID: "", resp: &ec2.DescribeTagsOutput{Tags: emptyTags}, expectedErr: true},
		{instanceID: "", resp: nil, expectedErr: true},
	}

	for _, c := range checks {
		c := c
		t.Run(fmt.Sprintf("Mock: Response: %#v expects %t", c.resp, c.expectedErr), func(t *testing.T) {
			ec2Client := EC2{Client: EC2Mock{Resp: c.resp}}

			got, err := ec2Client.GetAsgName(c.instanceID)

			fmt.Println("Got: ", got)
			if c.expectedErr {
				if err == nil {
					t.Errorf("An error is expected")
				}
			} else {
				if err != nil {
					t.Error("An error is not expected", err)
				}

				if got != asgName {
					t.Errorf("got != expected")
				}
			}
		})
	}
}
