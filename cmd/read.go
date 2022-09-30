package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	. "github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Retrieve and display the content in the Log Stream",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRead()
	},
}

func init() {
	rootCmd.AddCommand(readCmd)
}

var iconSelect = promptui.Styler(promptui.FGCyan)(promptui.IconSelect)

func executeRead() error {
	ctx := context.Background()

	// init cwl client
	client, err := newClient(ctx)
	if err != nil {
		return err
	}

	// get resources grouped by service
	resourceMap, err := getResourceMap(ctx, client)
	if err != nil {
		return err
	}

	// prompt: log group
	logGroup, err := promptLogGroup(resourceMap)
	if err != nil {
		return err
	}

	// get log streams by log group
	logStreams, err := getLogStreams(ctx, client, logGroup)
	if err != nil {
		return err
	}

	// prompt: log stream
	selStream, err := promptLogStream(logStreams)
	if err != nil {
		return err
	}

	// display logs
	if err := displayLogs(ctx, client, logGroup, selStream); err != nil {
		return err
	}

	return nil
}

func promptLogGroup(resourceMap ResourceMap) (string, error) {
	// prompt: 1/2
	tmpl1 := &promptui.SelectTemplates{
		Label:  "Select Log Group - 1/2",
		Active: fmt.Sprintf(`%s {{ if eq . ""}}{{ "others" | underline | cyan }}{{ else }}{{ . | underline | cyan }}{{ end }}`, iconSelect),
		// Inactive: "  {{ . }}",
		Inactive: `  {{ if eq . ""}}others{{ else }}{{ . }}{{ end }}`,
		Selected: " ",
	}

	prompt1 := promptui.Select{
		Size:      10,
		Items:     resourceMap.Services(),
		Templates: tmpl1,
	}

	_, service, err := prompt1.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	// prompt: 2/2
	items := resourceMap[service]

	tmpl2 := &promptui.SelectTemplates{
		Label:    "Select Log Group - 2/2",
		Active:   fmt.Sprintf("%s {{ . | underline | cyan }}", iconSelect),
		Inactive: "  {{ . }}",
		Selected: `{{ "Log Group:" | faint }}	` + service + `{{ . }}`,
	}

	searcher := func(input string, index int) bool {
		item := items[index]

		label := strings.ToLower(item)
		search := strings.ToLower(input)

		return strings.Contains(label, search)
	}

	prompt := promptui.Select{
		Size:      10,
		Items:     items,
		Templates: tmpl2,
		Searcher:  searcher,
	}

	_, resource, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	lg := service + resource

	return lg, nil
}

func promptLogStream(items []LogStream) (string, error) {
	tmpl := &promptui.SelectTemplates{
		Label:    "Select Log Stream",
		Active:   fmt.Sprintf(`%s {{ .Name | underline | cyan }}{{ .Date.Format " - 15:04:05" | underline | cyan }}`, iconSelect),
		Inactive: `  {{ .Name }}{{ .Date.Format " - 15:04:05" }}`,
		Selected: `{{ "Log Stream:" | faint }}	{{ .Name }}`,
	}

	searcher := func(input string, index int) bool {
		item := items[index]

		s := strings.ToLower(item.Name) + item.Date.Format(time.RFC3339)
		label := strings.ReplaceAll(s, "/", "")
		search := strings.ToLower(input)

		return strings.Contains(label, search)
	}

	prompt := promptui.Select{
		Size:      10,
		Items:     items,
		Templates: tmpl,
		Searcher:  searcher,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	return items[idx].Name, nil
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

// ResourceMap is a list of resources grouped by service
type ResourceMap map[string][]string

// Services return all the services in the struct
func (r ResourceMap) Services() []string {
	var ss []string
	for k := range r {
		ss = append(ss, k)
	}

	sort.Strings(ss)

	if ss[0] == "" {
		// move first item to the back
		s := ss[0]

		ss = ss[1:]
		ss = append(ss, s)
	}

	return ss
}

// getResourceMap retrieves and group CloudWatch Logs by services
func getResourceMap(ctx context.Context, client *cloudwatchlogs.Client) (ResourceMap, error) {
	rm := make(ResourceMap, 0)

	var nextToken *string
	for {
		// retrieve log groups
		out, err := client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
			NextToken: nextToken,
			// LogGroupNamePrefix: aws.String("/aws/lambda/exitpass"),
		})
		if err != nil {
			return nil, err
		}

		nextToken = out.NextToken

		for _, it := range out.LogGroups {
			// attempt to group resources base on service in best effort
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

	var service string
	if str := s[:idx+1]; str != "" {
		service = str
	}
	resource := s[idx+1:]

	return service, resource
}

type LogStream struct {
	Name string
	Date time.Time
}

func getLogStreams(ctx context.Context, client *cloudwatchlogs.Client, logGroup string) ([]LogStream, error) {
	out, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroup),
		Descending:   aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	var ls []LogStream
	for _, it := range out.LogStreams {
		ls = append(ls, LogStream{
			Name: *it.LogStreamName,
			Date: time.UnixMilli(*it.LastEventTimestamp).Local(),
		})
	}

	return ls, nil
}
