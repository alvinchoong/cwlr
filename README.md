# cwlr

cwlr is a simple CLI (Command Line Interface) application written in Go to interacts with AWS CloudWatch Logs easily

Uses:

- [cobra](github.com/spf13/cobra) for creating powerful modern CLI applications
- [promptui](github.com/manifoldco/promptui) for CLI interactions
- [aws-sdk](github.com/aws/aws-sdk-go-v2) to interact with AWS Services

## Pre-requisites

Install Go in 1.18 version minimum.

## Install the app

```shell
$ go install github.com/alvinchoong/cwlr@latest
```

## Run the app

```shell
$ cwlr

CLI tool for interacting with AWS CloudWatch Logs

Usage:
  cwlr [command]

Available Commands:
  help        Help about any command
  read        Retrieve and display the content in the Log Stream

Flags:
  -h, --help   help for cwlr

Use "cwlr [command] --help" for more information about a command.
```
