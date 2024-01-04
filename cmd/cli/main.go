package main

import (
	"fmt"
	broccli "github.com/mikogs/go-broccli/v2"
	"os"

	"github.com/Cardinal-Cryptography/github-actions-validator/pkg/dotgithub"
)

func main() {
	cli := broccli.NewCLI("github-actions-validator", "Validates GitHub Actions' .github directory", "Mikolaj Gasior <mikolaj@gasior.dev>")
	cmd := cli.AddCmd("validate", "Runs the validation on files from a specified directory", validateHandler)
	cmd.AddFlag("path", "p", "", "Path to .github directory", broccli.TypePathFile, broccli.IsDirectory|broccli.IsExistent|broccli.IsRequired)
	cmd.AddFlag("vars-file", "z", "", "Check if variable names exist in this file (one per line)", broccli.TypePathFile, broccli.IsExistent|broccli.IsRegularFile)
	cmd.AddFlag("secrets-file", "s", "", "Check if secret names exist in this file (one per line)", broccli.TypePathFile, broccli.IsExistent|broccli.IsRegularFile)
	_ = cli.AddCmd("version", "Prints version", versionHandler)
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		os.Args = []string{"App", "version"}
	}
	os.Exit(cli.Run())
}

func versionHandler(c *broccli.CLI) int {
	fmt.Fprintf(os.Stdout, VERSION+"\n")
	return 0
}

func validateHandler(c *broccli.CLI) int {
	// , c.Flag("vars-file"), c.Flag("secrets-file")
	err := dotGithub.NewFromPath(c.Flag("path"))
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
