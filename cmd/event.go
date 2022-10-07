/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// eventCmd represents the event command
var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Enable / disable event rules",
	RunE:  executeEvent,
}

func executeEvent(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// init eb client
	client, err := newEventBridgeClient(ctx)
	if err != nil {
		return err
	}

	// retrieve event rules
	rules, err := getEventRules(ctx, client)
	if err != nil {
		return err
	}

	// rule, err := promptEventRule(rules)
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("%+v\n", rule)
	for _, it := range rules {
		fmt.Printf("Name: %+v\n", *it.Name)
		if it.Description != nil {
			fmt.Printf("Desc: %+v\n", *it.Description)
		}
		if it.ScheduleExpression != nil {
			fmt.Printf("Expr: %+v\n", *it.ScheduleExpression)
		}
		fmt.Printf("State: %+v\n", it.State)
		fmt.Println("----")
	}

	return nil
}

func promptEventRule(rules []types.Rule) (string, error) {
	tmpl := &promptui.SelectTemplates{
		Label:    "Select Event Rule",
		Active:   fmt.Sprintf("%s {{ .Name | underline | cyan }}", iconSelect),
		Inactive: "  {{ . }}",
		Selected: `{{ "Event Rule:" | faint }}	{{ . }}`,
		Details:  ``,
	}

	// searcher := func(input string, index int) bool {
	// 	item := logGroups[index]

	// 	label := strings.ToLower(item)
	// 	search := strings.ToLower(input)

	// 	return strings.Contains(label, search)
	// }

	prompt := promptui.Select{
		Size:      10,
		Items:     rules,
		Templates: tmpl,
		// Searcher:  searcher,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	return result, nil
}

// types.RuleStateEnabled

type Rule struct {
	Name        string
	Description *string
	Expression  *string
	Enabled     bool
}

func getEventRules(ctx context.Context, client *eventbridge.Client) ([]types.Rule, error) {
	var rules []Rule

	var next *string
	for {
		out, err := client.ListRules(ctx, &eventbridge.ListRulesInput{})
		if err != nil {
			return nil, err
		}

		for _, it := range out.Rules {
			rule := Rule{}
			rules = append(rules, it)
		}

		next = out.NextToken
		if next == nil {
			break
		}
	}

	return rules, nil
}

// newEventBridgeClient attempts to create a new AWS Event Bridge Client
func newEventBridgeClient(ctx context.Context, opts ...func(*config.LoadOptions) error) (*eventbridge.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return eventbridge.NewFromConfig(cfg), nil
}

func init() {
	rootCmd.AddCommand(eventCmd)
}
