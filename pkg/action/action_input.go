package action

import (
	"fmt"
	"regexp"
)

type ActionInput struct {
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
	Required    bool   `yaml:"required"`
}

func (ai *ActionInput) Validate(action string, name string) ([]string, error) {
	var validationErrors []string
	m, err := regexp.MatchString(`^[a-z0-9][a-z0-9\-]+$`, name)
	if err != nil {
		return validationErrors, err
	}
	if !m {
		validationErrors = append(validationErrors, ai.formatError(action, name, "NA301", "Action input name should contain lowercase alphanumeric characters and hyphens only"))
	}

	if ai.Description == "" {
		validationErrors = append(validationErrors, ai.formatError(action, name, "NA302", "Action input must have a description"))
	}
	return validationErrors, nil
}

func (ai *ActionInput) formatError(action string, input string, code string, desc string) string {
	return fmt.Sprintf("%s: %-60s %s", code, "action "+action+" input "+input, desc)
}
