package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func main() {
	// This script is hard coded to check for logs within a test environment.
	// To use this script what needs to be changed are these two variables.
	// In AWS CloudWatch Logs, there are 'Log groups' as well as 'Log streams' within these groups.
	// Change these strings to your desired log group name and log stream name.
	var log_group_name = "testing_group"
	var log_stream_name = "testing_stream"

	// Initializes a connection to AWS
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-north-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create a variable that will be used to make requests to AWS CloudWatch
	client := cloudwatchlogs.NewFromConfig(cfg)

	// Requests to AWS CloudWatch to get all log events from the specified log group and stream
	resp, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(log_group_name),
		LogStreamName: aws.String(log_stream_name),
	})
	if err != nil {
		log.Fatalf("Failed to fetch log events: %v", err)
	}

	// Prints all the output from the response given by CloudWatch
	fmt.Println("Response")
	for _, event := range resp.Events {
		var timestamp = time.Unix(0, *event.Timestamp*int64(time.Millisecond))
		fmt.Println("\nTimestamp:", timestamp.Format(time.RFC3339))
		fmt.Println("Message:", *event.Message)
	}
}
