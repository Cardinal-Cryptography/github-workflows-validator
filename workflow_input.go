package main

import (
	"fmt"
	"regexp"
)

type WorkflowInput struct {
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
	Required    bool   `yaml:"required"`
}

func (wi *WorkflowInput) Validate(workflow string, placement string, name string) ([]string, error) {
	var validationErrors []string
	m, err := regexp.MatchString(`^[a-z0-9][a-z0-9\-]+$`, name)
	if err != nil {
		return validationErrors, err
	}
	if !m {
		validationErrors = append(validationErrors, wi.formatError(workflow, placement, name, "EW108", "Workflow input name should contain lowercase alphanumeric characters and hyphens only", "workflow-input-lowercase-alphanumeric-and-hyphens"))
	}

	if wi.Description == "" {
		validationErrors = append(validationErrors, wi.formatError(workflow, placement, name, "EW109", "Workflow input must have a description", "workflow-input-description-empty"))
	}
	return validationErrors, nil
}

func (wi *WorkflowInput) formatError(workflow string, placement string, input string, code string, desc string, name string) string {
	return fmt.Sprintf("%s: %-80s %s (%s)", code, "workflow "+workflow+" "+placement+" input "+input, desc, name)
}
