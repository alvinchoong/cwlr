package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search and display logs that matches the filter pattern or string",
	RunE:  excecuteSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func excecuteSearch(cmd *cobra.Command, args []string) error {
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

	// prompt: filter pattern
	pattern, err := promptPattern()
	if err != nil {
		return err
	}

	// prompt: start date
	start, err := promptDateTime("Start")
	if err != nil {
		return err
	}

	// prompt: end date
	end, err := promptDateTime("End")
	if err != nil {
		return err
	}

	// query
	logs, err := getFilteredLogs(ctx, client, selLogGroup, pattern, start, end)
	if err != nil {
		return err
	}

	// display
	for _, it := range logs {
		print(*it.Message, *it.Timestamp)
	}

	return nil
}

func promptPattern() (string, error) {
	prompt := promptui.Prompt{
		Label: "Filter Pattern",
	}

	return prompt.Run()
}

func promptDateTime(labelPrefix string) (*int64, error) {
	validateDate := func(input string) error {
		s := strings.TrimSpace(input)
		if len(s) == 0 {
			return nil
		}

		if _, err := time.Parse("2006-01-02", s); err != nil {
			return errors.New("invalid date format")
		}

		return nil
	}

	promptDate := promptui.Prompt{
		Label:    labelPrefix + " Date (YYYY-MM-DD)",
		Validate: validateDate,
	}

	resultDate, err := promptDate.Run()
	if err != nil {
		return nil, err
	}
	dateString := strings.TrimSpace(resultDate)

	if len(dateString) == 0 {
		return nil, nil
	}

	validateTime := func(input string) error {
		s := strings.TrimSpace(input)
		if _, err := time.Parse("15:04:05", s); err != nil {
			return errors.New("invalid time format")
		}

		return nil
	}

	promptTime := promptui.Prompt{
		Label:    labelPrefix + " Time (HH:MM:SS)",
		Validate: validateTime,
	}

	resultTime, err := promptTime.Run()
	if err != nil {
		return nil, err
	}
	timeString := strings.TrimSpace(resultTime)

	// parse and convert into unix time
	d, _ := time.Parse(time.RFC3339, fmt.Sprintf("%sT%sZ", dateString, timeString))
	m := d.UnixMilli()

	return &m, nil
}

func getFilteredLogs(ctx context.Context, client *cloudwatchlogs.Client, logGroup, pattern string, start, end *int64) ([]types.FilteredLogEvent, error) {
	// TODO: consider handling of pagination from CLI instead (e.g prompt for "more")

	var logs []types.FilteredLogEvent

	var next *string
	for {
		out, err := client.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName:  aws.String(logGroup),
			FilterPattern: aws.String(pattern),
			StartTime:     start,
			EndTime:       end,
			NextToken:     next,
		})
		if err != nil {
			return nil, err
		}

		logs = append(logs, out.Events...)

		next = out.NextToken
		if next == nil {
			break
		}
	}

	return logs, nil
}
