package main

import (
	"fmt"
	"regexp"
)

type ActionOutput struct {
	Description string `yaml:"description"`
	Value       string `yaml:"value"`
}

func (ao *ActionOutput) Validate(action string, name string) ([]string, error) {
	var validationErrors []string
	m, err := regexp.MatchString(`^[a-z0-9][a-z0-9\-]+$`, name)
	if err != nil {
		return validationErrors, err
	}
	if !m {
		validationErrors = append(validationErrors, ao.formatError(action, name, "NA501", "Action output name should contain lowercase alphanumeric characters and hyphens only"))
	}

	if ao.Description == "" {
		validationErrors = append(validationErrors, ao.formatError(action, name, "NA502", "Action output must have a description"))
	}
	return validationErrors, nil
}

func (ao *ActionOutput) formatError(action string, output string, code string, desc string) string {
	return fmt.Sprintf("%s: %-60s %s", code, "action "+action+" output "+output, desc)
}
