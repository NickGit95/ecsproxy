package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type ecsClient struct {
	ecs *ecs.Client
}

// Call the list tasks function to retrieve the task arns
func (e *ecsClient) listTasks(cluster *string) ([]string, error) {
	params := &ecs.ListTasksInput{
		Cluster: cluster,
	}
	tasks, err := e.ecs.ListTasks(context.TODO(), params)
	if err != nil {
		return nil, err
	}
	return tasks.TaskArns, nil
}

// Call the DescribeTaskDefinition function to retreive a task definition object
// based on a ARN
func (e *ecsClient) describeTasks(arns []string, cluster *string) ([]types.Task, error) {
	params := &ecs.DescribeTasksInput{
		Tasks:   arns,
		Cluster: cluster,
	}
	resp, err := e.ecs.DescribeTasks(context.TODO(), params)
	if err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// Call the DescribeTaskDefinition function to retreive a task definition object
// based on a ARN
func (e *ecsClient) describeTaskDefinition(arn *string) (*types.TaskDefinition, error) {
	params := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: arn,
	}
	resp, err := e.ecs.DescribeTaskDefinition(context.TODO(), params)
	if err != nil {
		return nil, err
	}
	return resp.TaskDefinition, nil
}

// List container instances from the cluster
func (e *ecsClient) listContainerInstances(cluster *string) ([]string, error) {
	params := &ecs.ListContainerInstancesInput{
		Cluster: cluster,
	}
	resp, err := e.ecs.ListContainerInstances(context.TODO(), params)
	if err != nil {
		return nil, err
	}
	return resp.ContainerInstanceArns, nil
}

// Describe the EC2 container instances on a cluster
func (e *ecsClient) describeContainerInstances(
	arns []string,
	cluster *string,
) ([]types.ContainerInstance, error) {
	params := &ecs.DescribeContainerInstancesInput{
		ContainerInstances: arns,
		Cluster:            cluster,
	}
	resp, err := e.ecs.DescribeContainerInstances(context.TODO(), params)
	if err != nil {
		return nil, err
	}
	return resp.ContainerInstances, err
}
