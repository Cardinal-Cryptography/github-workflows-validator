package main

import (
	"fmt"
	gocli "github.com/nicholasgasior/go-broccli"
	"os"

	"github.com/Cardinal-Cryptography/github-actions-validator/pkg/dotgithub"
)

func main() {
	cli := gocli.NewCLI("github-actions-validator", "Validates GitHub Actions' .github directory", "devops@cc")
	cmdValidate := cli.AddCmd("validate", "Runs the validation on files from a specified directory", validateHandler)
	cmdValidate.AddFlag("path", "p", "", "Path to .github directory", gocli.TypePathFile, gocli.IsDirectory|gocli.IsExistent|gocli.IsRequired)
	cmdValidate.AddFlag("vars-file", "z", "", "Check if variable names exist in this file (one per line)", gocli.TypePathFile, gocli.IsExistent)
	cmdValidate.AddFlag("secrets-file", "s", "", "Check if secret names exist in this file (one per line)", gocli.TypePathFile, gocli.IsExistent)
	_ = cli.AddCmd("version", "Prints version", versionHandler)
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		os.Args = []string{"App", "version"}
	}
	os.Exit(cli.Run())
}

func versionHandler(c *gocli.CLI) int {
	fmt.Fprintf(os.Stdout, VERSION+"\n")
	return 0
}

func validateHandler(c *gocli.CLI) int {
	dotGithub := dotgithub.DotGithub{
		Path:        c.Flag("path"),
		VarsFile:    c.Flag("vars-file"),
		SecretsFile: c.Flag("secrets-file"),
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
