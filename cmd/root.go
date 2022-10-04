/*
Copyright © 2022 Alvin Choong

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	. "github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cwlr",
	Short: "CLI tool for interacting with AWS CloudWatch Logs",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// newClient attempts to create a new AWS Cloudwatch Logs Client
func newClient(ctx context.Context, opts ...func(*config.LoadOptions) error) (*cloudwatchlogs.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return cloudwatchlogs.NewFromConfig(cfg), nil
}

func print(msg string, milli int64) {
	dt := time.UnixMilli(milli).Format(time.RFC3339)

	fmt.Printf("%s: %s", Cyan(dt), Green(msg))
}
