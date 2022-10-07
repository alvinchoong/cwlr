/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
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

	rule, err := promptEventRule(rules)
	if err != nil {
		return err
	}

	fmt.Printf("selected: %+v\n", rule)

	return nil
}

func promptEventRule(rules []Rule) (string, error) {
	tmpl := &promptui.SelectTemplates{
		Label:    "Select Event Rule",
		Active:   fmt.Sprintf("%s {{ .Name | underline | cyan }}", iconSelect),
		Inactive: "  {{ .Name }}",
		Selected: `{{ "Event Rule:" | faint }}	{{ .Name }}`,
		Details: `
--------- Details ----------
{{ "Name:" | faint }}	{{ .Name }}
{{ "Description:" | faint }}	{{ .Description }}
{{ "ScheduleExpression:" | faint }}	{{ .ScheduleExpression }}
{{ "State:" | faint }}	{{if eq .State "ENABLED" }}{{ .State | green }}{{else}}{{ .State | red }}{{end}}`,
	}

	prompt := promptui.Select{
		Size:      10,
		Items:     rules,
		Templates: tmpl,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	return result, nil
}

type Rule struct {
	Name               string
	Description        *string
	ScheduleExpression *string
	State              string
}

func getEventRules(ctx context.Context, client *eventbridge.Client) ([]Rule, error) {
	var rules []Rule

	var next *string
	for {
		out, err := client.ListRules(ctx, &eventbridge.ListRulesInput{})
		if err != nil {
			return nil, err
		}

		for _, it := range out.Rules {
			rule := Rule{
				Name:               *it.Name,
				Description:        it.Description,
				ScheduleExpression: it.ScheduleExpression,
				State:              string(it.State),
			}

			rules = append(rules, rule)
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
