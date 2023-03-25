package main

import (
	"fmt"
	gocli "github.com/mikogs/lib-go-cli"
	"os"
)

func main() {
	cli := gocli.NewCLI("github-actions-validator", "Validates GitHub Actions' .github directory", "Mikolaj Gasior <miko@gen64.net>")
	cmdValidate := cli.AddCmd("validate", "Runs the validation on files from a specified directory", validateHandler)
	cmdValidate.AddFlag("path", "p", "", "Path to .github directory", gocli.TypePathDir|gocli.MustExist|gocli.Required, nil)
	_ = cli.AddCmd("version", "Prints version", versionHandler)
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		os.Args = []string{"App", "version"}
	}
	os.Exit(cli.Run(os.Stdout, os.Stderr))
}

func versionHandler(c *gocli.CLI) int {
	fmt.Fprintf(os.Stdout, VERSION+"\n")
	return 0
}

func validateHandler(c *gocli.CLI) int {
	dotGithub := DotGithub{
		Path: c.Flag("path"),
	}
	err := dotGithub.InitFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!!! Error with initialization: %s\n", err.Error())
		return 1
	}
	validationErrors, err := dotGithub.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!!! Error with validation: %s\n", err.Error())
		return 1
	}
	for _, verr := range validationErrors {
		fmt.Fprintf(os.Stdout, "%s\n", verr)
	}
	return 0
}
