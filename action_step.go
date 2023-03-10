package main

import (
	"fmt"
	"regexp"
	"strings"
)

type ActionStep struct {
	Name  string            `yaml:"name"`
	Id    string            `yaml:"id"`
	Uses  string            `yaml:"uses"`
	Shell string            `yaml:"bash"`
	Env   map[string]string `yaml:"env"`
	Run   string            `yaml:"run"`
	With  map[string]string `yaml:"with"`
}

func (as *ActionStep) Validate(action string, name string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	if as.Uses != "" && strings.HasPrefix(as.Uses, "./.github/") {
		verrs, err := as.validateUsesLocalAction(action, name, as.Uses, d)
		if err != nil {
			return validationErrors, err
		}
		if len(verrs) > 0 {
			for _, verr := range verrs {
				validationErrors = append(validationErrors, verr)
			}
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) validateUsesLocalAction(action string, step string, uses string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	m, err := regexp.MatchString(`^\.\/\.github\/actions\/[a-z0-9\-]+$`, uses)
	if err != nil {
		return validationErrors, err
	}
	if !m {
		validationErrors = append(validationErrors, as.formatError(action, step, "EA111", fmt.Sprintf("Path to local action '%s' is invalid", uses), "action-step-uses-invalid-local-path"))
	}

	usedAction := strings.Replace(uses, "./.github/actions/", "", -1)
	if d.Actions == nil || d.Actions[usedAction] == nil {
		validationErrors = append(validationErrors, as.formatError(action, step, "EA112", fmt.Sprintf("Call to non-existing action '%s'", uses), "action-step-uses-nonexisting-local-action"))
		return validationErrors, nil
	}

	if d.Actions[usedAction].Inputs != nil {
		for dainputName, daInput := range d.Actions[usedAction].Inputs {
			if daInput.Required {
				if as.With == nil || as.With[dainputName] == "" {
					validationErrors = append(validationErrors, as.formatError(action, step, "EA113", fmt.Sprintf("Required input '%s' missing for action '%s'", dainputName, uses), "action-step-uses-local-action-missing-input"))
				}
			}
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) formatError(action, step string, code string, desc string, name string) string {
	return fmt.Sprintf("%s: %-60s %s (%s)", code, "action "+action+" step "+step, desc, name)
}
