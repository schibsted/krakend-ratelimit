package rate_limit

import (
	"github.com/devopsfaith/krakend/logging"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	gate_aws "github.com/tgracchus/krakend-ratelimit/aws"
)

func NewAwsNodeCounter(EC2 *gate_aws.EC2, autoScaling *gate_aws.AutoScaling,
	iid *ec2metadata.EC2InstanceIdentityDocument, logger logging.Logger) NodeCounter {
	lastNumber := 3
	return func() int {
		autoScalingName, err := EC2.GetAsgName(iid.InstanceID)
		if err != nil {
			return lastNumber
		}

		asgNames := []*string{aws.String(autoScalingName)}
		result, err := autoScaling.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: asgNames})
		if err != nil {
			return lastNumber
		}

		if len(result.AutoScalingGroups) == 1 {
			logger.Debug("Instances: ", len(result.AutoScalingGroups[0].Instances))
			nInstances := len(result.AutoScalingGroups[0].Instances)
			lastNumber = nInstances
			return nInstances
		}

		return lastNumber
	}

}
