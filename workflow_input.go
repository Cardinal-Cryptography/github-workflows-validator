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
		validationErrors = append(validationErrors, wi.formatError(workflow, placement, name, "NW301", "Workflow input name should contain lowercase alphanumeric characters and hyphens only"))
	}

	if wi.Description == "" {
		validationErrors = append(validationErrors, wi.formatError(workflow, placement, name, "NW302", "Workflow input must have a description"))
	}
	return validationErrors, nil
}

func (wi *WorkflowInput) formatError(workflow string, placement string, input string, code string, desc string) string {
	return fmt.Sprintf("%s: %-80s %s", code, "workflow "+workflow+" "+placement+" input "+input, desc)
}
