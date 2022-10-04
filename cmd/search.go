package cmd

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
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

	// TODO: add prompt for start / end time

	out, err := client.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  aws.String(selLogGroup),
		FilterPattern: aws.String(pattern),
	})
	if err != nil {
		return err
	}

	for _, it := range out.Events {
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
