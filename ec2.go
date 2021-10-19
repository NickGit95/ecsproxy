package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type ec2Client struct {
	ec2 *ec2.Client
}

func (e *ec2Client) describeInstance(id string) (types.Instance, error) {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []string{
			id,
		},
	}
	reservations, err := e.ec2.DescribeInstances(context.TODO(), params)
	if err != nil {
		return types.Instance{}, err
	}
	return reservations.Reservations[0].Instances[0], err
}
