package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/manifoldco/promptui"

	. "github.com/logrusorgru/aurora"
)

type ResourceMap map[string][]string

func main() {
	ctx := context.Background()

	// init client
	client, err := newCloudwatchLogsClient(ctx)
	check(err)

	// get resources grouped by service
	resourceMap, err := getResourceMap(ctx, client)
	check(err)

	// prompt: service
	selService, err := promptSelect("Select Service", resourceMap.Services())
	check(err)

	// prompt: resource
	resources := resourceMap[selService]
	selResource, err := promptSelect("Select Resource", resources)
	check(err)

	logGroup := selService + selResource

	// get log streams by log group
	streams, err := getLogStreams(ctx, client, logGroup)
	check(err)

	// prompt: log stream
	selLog, err := promptSelect("Select Log Stream", streams)
	check(err)

	err = displayLogs(ctx, client, logGroup, selLog)
	check(err)
}

// newCloudwatchLogsClient attempts to create a new AWS Cloudwatch Logs Client
func newCloudwatchLogsClient(ctx context.Context, opts ...func(*config.LoadOptions) error) (*cloudwatchlogs.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return cloudwatchlogs.NewFromConfig(cfg), nil
}

func promptSelect(label string, items []string) (string, error) {
	searcher := func(input string, index int) bool {
		item := items[index]

		name := strings.ToLower(item)
		search := strings.ToLower(input)

		return strings.Contains(name, search)
	}

	prompt := promptui.Select{
		Label:    label,
		Items:    items,
		Searcher: searcher,
		Size:     10,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	return result, nil
}

func getResourceMap(ctx context.Context, client *cloudwatchlogs.Client) (ResourceMap, error) {
	rm := make(ResourceMap, 0)

	var nextToken *string
	for {
		// retrieve log groups
		out, err := client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		nextToken = out.NextToken

		for _, it := range out.LogGroups {
			// group resources base on service
			s, r := split(*it.LogGroupName)

			if _, ok := rm[s]; !ok {
				rm[s] = []string{}
			}

			rm[s] = append(rm[s], r)
		}

		if nextToken == nil {
			break
		}
	}

	return rm, nil
}

func split(s string) (string, string) {
	n := 1
	if strings.HasPrefix(s, "/aws/") {
		n = 5
	}

	idx := strings.Index(s[n:], "/")
	if idx > -1 {
		idx += n
	}

	service := "/"
	if str := s[:idx+1]; str != "" {
		service = str
	}
	resource := s[idx+1:]

	return service, resource
}

func (r ResourceMap) Services() []string {
	var s []string
	for k := range r {
		s = append(s, k)
	}

	return s
}

func getLogStreams(ctx context.Context, client *cloudwatchlogs.Client, logGroup string) ([]string, error) {
	out, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroup),
		Descending:   aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	var s []string
	for _, it := range out.LogStreams {
		s = append(s, *it.LogStreamName)
	}

	return s, nil
}

func displayLogs(ctx context.Context, client *cloudwatchlogs.Client, logGroup, logStream string) error {
	out, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &logGroup,
		LogStreamName: &logStream,
	})
	if err != nil {
		return err
	}

	for _, it := range out.Events {
		dt := time.UnixMilli(*it.Timestamp).Format(time.RFC3339)

		fmt.Printf("%s: %s", Cyan(dt), Green(*it.Message))
	}

	return nil
}

func check(e error) {
	if e != nil {
		fmt.Println(e.Error())
		panic(e)
	}
}
