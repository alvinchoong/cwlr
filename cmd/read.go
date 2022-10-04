package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Retrieve and display the content in the Log Stream",
	RunE:  executeRead,
}

func init() {
	rootCmd.AddCommand(readCmd)
}

var iconSelect = promptui.Styler(promptui.FGCyan)(promptui.IconSelect)

func executeRead(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// init cwl client
	client, err := newClient(ctx)
	if err != nil {
		return err
	}

	// get cloudwatch log groups
	logGroups, err := getLogGroups(ctx, client)
	if err != nil {
		return err
	}

	// prompt: log group
	var selLogGroup string
	if FlagGroup {
		selLogGroup, err = promptLogGroupWithGrouping(logGroups)
	} else {
		selLogGroup, err = promptLogGroup(logGroups)
	}
	if err != nil {
		return err
	}

	// get log streams by log group
	logStreams, err := getLogStreams(ctx, client, selLogGroup)
	if err != nil {
		return err
	}

	// prompt: log stream
	selStream, err := promptLogStream(logStreams)
	if err != nil {
		return err
	}

	// query
	logs, err := getLogs(ctx, client, selLogGroup, selStream)
	if err != nil {
		return err
	}

	// display
	for _, it := range logs {
		print(*it.Message, *it.Timestamp)
	}

	return nil
}

func promptLogGroup(logGroups []string) (string, error) {
	tmpl := &promptui.SelectTemplates{
		Label:    "Select Log Group",
		Active:   fmt.Sprintf("%s {{ . | underline | cyan }}", iconSelect),
		Inactive: "  {{ . }}",
		Selected: `{{ "Log Group:" | faint }}	{{ . }}`,
	}

	searcher := func(input string, index int) bool {
		item := logGroups[index]

		label := strings.ToLower(item)
		search := strings.ToLower(input)

		return strings.Contains(label, search)
	}

	prompt := promptui.Select{
		Size:      10,
		Items:     logGroups,
		Templates: tmpl,
		Searcher:  searcher,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	return result, nil
}

func promptLogGroupWithGrouping(logGroups []string) (string, error) {
	resourceMap := toResourceMap(logGroups)

	// prompt: 1/2
	tmpl1 := &promptui.SelectTemplates{
		Label:    "Select Log Group - 1/2",
		Active:   fmt.Sprintf(`%s {{ if eq . ""}}{{ "others" | underline | cyan }}{{ else }}{{ . | underline | cyan }}{{ end }}`, iconSelect),
		Inactive: `  {{ if eq . ""}}others{{ else }}{{ . }}{{ end }}`,
	}

	prompt1 := promptui.Select{
		Size:         10,
		Items:        resourceMap.Services(),
		Templates:    tmpl1,
		HideSelected: true,
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

// toResourceMap group CloudWatch Logs by services
func toResourceMap(logGroups []string) ResourceMap {
	rm := make(ResourceMap, 0)

	for _, it := range logGroups {
		// best effort grouping of resources
		n := 1
		if strings.HasPrefix(it, "/aws/") {
			n = 5
		}

		idx := strings.Index(it[n:], "/")
		if idx > -1 {
			idx += n
		}

		var service string
		if str := it[:idx+1]; str != "" {
			service = str
		}
		resource := it[idx+1:]

		if _, ok := rm[service]; !ok {
			rm[service] = []string{}
		}

		rm[service] = append(rm[service], resource)
	}

	return rm
}

// getLogGroups retrieves all CloudWatch Logs
func getLogGroups(ctx context.Context, client *cloudwatchlogs.Client) ([]string, error) {
	var lg []string

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
			lg = append(lg, *it.LogGroupName)
		}

		if nextToken == nil {
			break
		}
	}

	return lg, nil
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

func getLogs(ctx context.Context, client *cloudwatchlogs.Client, logGroup, logStream string) ([]types.OutputLogEvent, error) {
	// TODO: consider handling of pagination from CLI instead (e.g prompt for "more")

	var logs []types.OutputLogEvent

	var next *string
	for {
		out, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroup,
			LogStreamName: &logStream,
			StartFromHead: aws.Bool(true),
			NextToken:     next,
		})
		if err != nil {
			return nil, err
		}

		if next != nil && *next == *out.NextForwardToken {
			break
		}

		logs = append(logs, out.Events...)
		next = out.NextForwardToken
	}

	return logs, nil
}
