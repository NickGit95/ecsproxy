package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type scanner struct {
	ecsClient *ecsClient
	ec2Client *ec2Client
	ec2Map    *map[string]string
}

type ecsContainer struct {
	Name    string
	Host    string
	Port    int32
	Address string
}

// Get all tasks running on the container
func (s *scanner) scan(cluster *string) ([]*ecsContainer, error) {
	taskArns, err := s.ecsClient.listTasks(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks, %v", err)
	}
	tasks, err := s.ecsClient.describeTasks(taskArns, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to describe tasks, %v", err)
	}
	s.ec2Map, err = s.ec2Scan(cluster)
	if err != nil {
		return nil, err
	}
	return s.extractContainers(tasks), nil
}

// Declare a slice with container structs
func (s *scanner) extractContainers(tasks []types.Task) []*ecsContainer {
	containerSlice := make([]*ecsContainer, 0, 30)
	for _, task := range tasks {
		taskDefinition, err := s.ecsClient.describeTaskDefinition(task.TaskDefinitionArn)
		if err != nil {
			log.Printf("failed describe task definition, %v", err)
			continue
		}
		containerDef, err := extractEnvironment(taskDefinition, task.LaunchType)
		if err != nil {
			log.Println(err)
			continue
		}

		// Loop through all the containers to find one with the same name
		// as containerDef
		for _, container := range task.Containers {
			networks := container.NetworkInterfaces
			bindings := container.NetworkBindings
			if containerDef.Name == *container.Name {
				if len(networks) > 0 && task.LaunchType == "FARGATE" {
					containerDef.Address = *networks[0].PrivateIpv4Address
				} else if len(bindings) > 0 && task.LaunchType == "EC2" {
					containerDef.Address = (*s.ec2Map)[*task.ContainerInstanceArn]
					containerDef.Port = *container.NetworkBindings[0].HostPort
				}
				containerSlice = append(containerSlice, containerDef)
			}
		}
	}
	return containerSlice
}

// Extract ec2 address information
func (s *scanner) ec2Scan(cluster *string) (*map[string]string, error) {
	ec2Map := make(map[string]string)
	arns, err := s.ecsClient.listContainerInstances(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list container instances, %v", err)
	}
	if len(arns) == 0 {
		log.Println("no container instances found in cluster. Skipping EC2 scan")
		return &ec2Map, nil
	}
	instances, err := s.ecsClient.describeContainerInstances(arns, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to describe container instances, %v", err)
	}
	for _, instance := range instances {
		instanceParams, err := s.ec2Client.describeInstance(*instance.Ec2InstanceId)
		if err != nil {
			return nil, fmt.Errorf("failed to describe container instance details, %v", err)
		}
		ec2Map[*instance.ContainerInstanceArn] = *instanceParams.PrivateIpAddress
	}
	return &ec2Map, nil
}

// Extract information from task definitions
func extractEnvironment(
	taskDef *types.TaskDefinition,
	launchType types.LaunchType,
) (*ecsContainer, error) {
	container := ecsContainer{}

	// Loop though each container definition to see if the host and port
	// variables are defined
	for _, containerDef := range taskDef.ContainerDefinitions {
		host, port := extractHostPort(containerDef)
		if host != "" {
			// If the port wasn't defined on the environments, then
			// extract it from the port mappings (only for FARGATE)
			if port == 0 && launchType == "FARGATE" {
				port = *containerDef.PortMappings[0].ContainerPort
			}
			container = ecsContainer{
				Name: *containerDef.Name,
				Host: host,
				Port: port,
			}
			return &container, nil
		}
	}
	return &container, fmt.Errorf(
		"VIRTUAL_HOST variable not found on %s task definition, skipping",
		*taskDef.Family,
	)
}

// Look into a container's variables to see if the HOST and PORT variables are defined
func extractHostPort(containerDef types.ContainerDefinition) (string, int32) {
	host := ""
	var port int32 = 0
	for _, environment := range containerDef.Environment {
		if *environment.Name == "VIRTUAL_HOST" {
			host = strings.ToLower(*environment.Value)
		} else if *environment.Name == "VIRTUAL_PORT" {
			portConv, err := strconv.ParseInt(*environment.Value, 10, 64)
			if err != nil {
				log.Printf("Error parsing VIRTUAL_PORT env variable.")
			} else {
				port = int32(portConv)
			}
		}
	}
	return host, port
}
