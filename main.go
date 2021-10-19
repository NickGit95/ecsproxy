package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

const (
	metadataVariable = "ECS_CONTAINER_METADATA_URI_V4"
)

var (
	cluster      string
	region       string
	outputFile   string
	templateFile string
	signal       string
	once         bool
	freq         int
)

type metadata struct {
	cluster string `json:"Cluster"`
}

func main() {
    // Init flags
    initFlags()

	// Check the region flag
	if region == "" {
		log.Fatalln("region flag not found. please use -r to define the cluster region")
	}

	// Check the cluster flag
	cluster, err := getCluster()
	if err != nil {
		log.Fatalln(err)
	}

	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create the ecs, ec2 and scanner structs
	clientEcs := ecsClient{
		ecs: ecs.NewFromConfig(cfg),
	}
	clientEc2 := ec2Client{
		ec2: ec2.NewFromConfig(cfg),
	}
	execute(clientEcs, clientEc2, cluster)
	if !once {
		time.Sleep(time.Duration(freq))
		for range time.Tick(time.Second * time.Duration(freq)) {
			execute(clientEcs, clientEc2, cluster)
		}
	}
}

// Initialize flags
func initFlags() {
	// Define flags
	flag.StringVar(&cluster, "cluster", os.Getenv("ECS_PROXY_CLUSTER"), "The cluster to scan.")
	flag.StringVar(&region, "region", os.Getenv("ECS_PROXY_REGION"), "The AWS region to use.")
	flag.StringVar(
		&templateFile,
		"template",
		"template.tmpl",
		"The template file to use for nginx configuration.",
	)
	flag.StringVar(
		&outputFile,
		"output",
		"/etc/nginx/conf.d/default.conf",
		"The output file for nginx configuration.",
	)
	flag.StringVar(
		&signal,
		"signal",
		"nginx -s reload",
		"Command to use for updating the nginx configuration.",
	)
	flag.BoolVar(&once, "once", false, "Add this flag to run the scan only once.")
	flag.IntVar(&freq, "freq", 30, "Time in secconds between each scan.")
	flag.Parse()
}

// Execute the scaner and update nginx
func execute(clientEcs ecsClient, clientEc2 ec2Client, cluster *string) {
	// Scan the cluster and get the list of containers
	scanner := scanner{
		ecsClient: &clientEcs,
		ec2Client: &clientEc2,
	}
	containers, err := scanner.scan(cluster)
	if err != nil {
		log.Fatalln(err)
	}
	writeTemplate(containers)
	runSignal()
}

// Check the metadata server if the clusterFlag was not set
func getCluster() (*string, error) {
	if cluster == "" {
		log.Println("using metadata server")
		if url, ok := os.LookupEnv(metadataVariable); ok {
			meta, err := http.Get(url + "/task")
			if err != nil {
				return &cluster, fmt.Errorf(
					"there was an error with the metadata server %s",
					err,
				)
			}
			var m metadata
			decoder := json.NewDecoder(meta.Body)
			err = decoder.Decode(&m)
			if err != nil {
				log.Panicln(err)
			}
			log.Printf("the cluster arn from metadata is: %s", m.cluster)
			return &m.cluster, nil
		}
		return nil, fmt.Errorf("metadata env variable not found. Are you running ouside of ecs?")
	}
	return &cluster, nil
}

// Write the nginx template
func writeTemplate(containers []*ecsContainer) {
	t, err := template.ParseFiles(templateFile)
	containerMap := getContainerMap(containers)
	if err != nil {
		log.Panicln(err)
	}
	f, err := os.Create(outputFile)
	if err != nil {
		log.Panicln(err)
	}
	defer f.Close()
	err = t.Execute(f, containerMap)
}

// Convert the container slice into a map of slices. This way we can implement
// proper load balancing on the nginx template
func getContainerMap(containers []*ecsContainer) map[string][]*ecsContainer {
	containerMap := make(map[string][]*ecsContainer)
	for _, v := range containers {
		if _, ok := containerMap[v.Name]; !ok {
			containerMap[v.Host] = make([]*ecsContainer, 0)
		}
		containerMap[v.Host] = append(containerMap[v.Host], v)
	}
	return containerMap
}

func runSignal() error {
	log.Printf("running signal command \"%s\"", signal)
	output, err := exec.Command("/bin/sh", "-c", signal).CombinedOutput()
	log.Println("===== output start =====")
	log.Println(string(output))
	log.Println("===== output end =====")
	return err
}
