package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	ctypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

func stop_ongoing_query(client *cloudwatchlogs.Client, query_id string) cloudwatchlogs.StopQueryOutput {
	// Debug function to test what happens if query unexpectedly where to cancel
	stop_query, err := client.StopQuery(context.TODO(), &cloudwatchlogs.StopQueryInput{
		QueryId: aws.String(query_id),
	})
	if err != nil {
		log.Fatalf("Failed to stop query: %v", err)
	}
	return *stop_query
}

func check_query(client *cloudwatchlogs.Client, query_id string) {
	// Creates a request to CloudWatch Logs to get the query result
	result, err := client.GetQueryResults(context.TODO(), &cloudwatchlogs.GetQueryResultsInput{
		QueryId: aws.String(query_id),
	})
	if err != nil {
		log.Fatalf("Failed to get the Query result: %v", err)
	}

	// Uncomment the line below to see the behavior by the script if the
	// Log Insights Query where to be cancelled
	// stop_ongoing_query(client, query_id)

	// This check looks for the status code: 'Completed' from StartQuery
	// Also creates a recursion to try again if the status code
	// comes back as 'Running'. As the code is now, the status 'Schedueled' will not be returned.
	if result.Status == ctypes.QueryStatusComplete {
		for _, value := range result.Results {
			response, err := client.GetLogRecord(context.TODO(), &cloudwatchlogs.GetLogRecordInput{
				LogRecordPointer: aws.String(*value[2].Value),
			})
			if err != nil {
				log.Fatalf("Failed to fetch log record: %v", err)
			}

			fmt.Printf("\nTimestamp: %v\n", *value[0].Value)
			fmt.Printf("Message: %v", response.LogRecord["@message"])
		}
	} else if result.Status == ctypes.QueryStatusRunning {

		check_query(client, query_id)
	} else {
		fmt.Printf("The status of this query is: %v\n", result.Status)
		log.Fatalf("Query ID: %v", query_id)
	}
}

func main() {
	// This script will fetch resources only in CloudWatch Logs with the 'filtered_service' variable.
	// It also uses tags in the form of key/value pairs to look for logs matching
	// those specific tags.
	// Bellow is the key and value(s) of these tags.
	tagKey := aws.String("another_new_tag")
	tagValues := []string{
		"another value",
	}
	tag_filter := types.TagFilter{}
	tag_filter.Key = tagKey
	tag_filter.Values = tagValues
	filtered_service := []string{"logs"}
	var log_group_identifiers []string
	var log_query_string = "fields @timestamp, @message, @logGroup | sort @timestamp desc | limit 20"
	var from_epoch_date = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	var query_start_date = time.Date(2023, 7, 1, 6, 30, 0, 0, time.UTC)
	var query_end_date = time.Now().UTC()
	var query_start_time = query_start_date.Sub(from_epoch_date)
	var query_end_time = query_end_date.Sub(from_epoch_date)

	// Initializes a connection to AWS
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-north-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create a configuration that will be used to make requests to AWS CloudWatch Logs API
	client := cloudwatchlogs.NewFromConfig(cfg)

	// Create a configuration that will be used to make requests to AWS Resource Groups Tagging API
	resource_client := resourcegroupstaggingapi.NewFromConfig(cfg)

	// Create a request to AWS Resource Groups Tagging API to get all resources
	tags, err := resource_client.GetResources(context.TODO(), &resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: []types.TagFilter{
			tag_filter,
		},
		ResourceTypeFilters: filtered_service,
	})
	if err != nil {
		log.Fatalf("Failed to get tags: %v", err)
	}

	// Iterates through the list given from the 'GetResources' call and appends each resource ARN to a list
	for _, thing := range tags.ResourceTagMappingList {
		log_group_identifiers = append(log_group_identifiers, *thing.ResourceARN)
	}

	// A form of debugging to the user which log group(s) will be queried
	fmt.Printf("A list of the Log Group identifiers filtered: \n%v\n", log_group_identifiers)

	// Initializes a query for CloudWatch Logs Insights by the use of the Log Group Identifiers from the ResourceTagMappingList
	if len(log_group_identifiers) < 1 {
		log.Fatalf("No log groups found with tag pattern: {%s: %v}", *tagKey, tagValues)
	}
	query, err := client.StartQuery(context.TODO(), &cloudwatchlogs.StartQueryInput{
		EndTime:             aws.Int64(int64(query_end_time.Seconds())),
		StartTime:           aws.Int64(int64(query_start_time.Seconds())),
		QueryString:         aws.String(log_query_string),
		LogGroupIdentifiers: log_group_identifiers,
	})
	if err != nil {
		log.Fatalf("Failed to start new Query: %v", err)
	}

	// Variable to store the Query ID
	var query_id = *query.QueryId

	// A message for the user to make sure the script works
	fmt.Println("\nWaiting for CloudWatch Logs Query results...")

	// Call to a function that returns the query result
	check_query(client, query_id)
}
