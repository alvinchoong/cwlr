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
	. "github.com/logrusorgru/aurora"
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

	// prompt: event rule
	rule, err := promptEventRule(rules)
	if err != nil {
		return err
	}

	// prompt: confirmation
	confirm, err := promptConfirmation(rule)
	if err != nil {
		return err
	}

	if !confirm {
		fmt.Println("exiting...")

		return nil
	}

	fmt.Printf("selected: %+v\n", rule)

	return nil
}

func promptEventRule(rules []Rule) (Rule, error) {
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

	idx, _, err := prompt.Run()
	if err != nil {
		return Rule{}, fmt.Errorf("prompt failed %v", err)
	}

	return rules[idx], nil
}

func promptConfirmation(rule Rule) (bool, error) {
	stateEnable := Green(types.RuleStateEnabled)
	stateDisable := Red(types.RuleStateDisabled)

	oldState, newState := stateDisable, stateEnable
	if rule.State == types.RuleStateEnabled {
		oldState, newState = stateEnable, stateDisable
	}

	tmpl := &promptui.SelectTemplates{
		Label:    fmt.Sprintf("Confirm to %s ?", Bold(newState)),
		Active:   fmt.Sprintf("%s {{ . | underline | cyan }}", iconSelect),
		Inactive: "  {{ . }}",
		Selected: fmt.Sprintf(`{{ "State Change:" | faint }}	%s to %s`, oldState, newState),
	}

	prompt := promptui.Select{
		Size:      10,
		Items:     []string{"Yes", "No"},
		Templates: tmpl,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return false, fmt.Errorf("prompt failed %v", err)
	}

	return result == "Yes", nil
}

type Rule struct {
	Name               string
	Description        *string
	ScheduleExpression *string
	State              types.RuleState
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
				State:              it.State,
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
