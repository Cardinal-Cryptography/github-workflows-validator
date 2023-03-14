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
	verrs, err := as.validateUses(action, name, as.Uses, d)
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = as.validateCalledStepOutputs(action, name, as.Uses, d)
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) validateUses(action string, name string, uses string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	if uses == "" {
		return validationErrors, nil
	}

	if strings.HasPrefix(as.Uses, "./.github/") {
		verrs, err := as.validateUsesLocalAction(action, name, as.Uses, d)
		if err != nil {
			return validationErrors, err
		}
		if len(verrs) > 0 {
			for _, verr := range verrs {
				validationErrors = append(validationErrors, verr)
			}
		}
	} else {
		m, err := regexp.MatchString(`[a-zA-Z0-9\-\_]+\/[a-zA-Z0-9\-\_]+@[a-zA-Z0-9\.\-\_]+`, as.Uses)
		if err != nil {
			return validationErrors, err
		}
		if !m {
			validationErrors = append(validationErrors, as.formatError(action, name, "EA113", fmt.Sprintf("Path to external action '%s' is invalid", as.Uses), "action-step-uses-invalid-external-path"))
		} else {
			verrs, err := as.validateUsesExternalAction(action, name, as.Uses, d)
			if err != nil {
				return validationErrors, err
			}
			if len(verrs) > 0 {
				for _, verr := range verrs {
					validationErrors = append(validationErrors, verr)
				}
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
		validationErrors = append(validationErrors, as.formatError(action, step, "EA112", fmt.Sprintf("Call to non-existing local action '%s'", uses), "action-step-uses-nonexisting-local-action"))
		return validationErrors, nil
	}

	if d.Actions[usedAction].Inputs != nil {
		for daInputName, daInput := range d.Actions[usedAction].Inputs {
			if daInput.Required {
				if as.With == nil || as.With[daInputName] == "" {
					validationErrors = append(validationErrors, as.formatError(action, step, "EA113", fmt.Sprintf("Required input '%s' missing for local action '%s'", daInputName, uses), "action-step-uses-local-action-missing-input"))
				}
			}
		}
	}
	if as.With != nil {
		for usedInput := range as.With {
			if d.Actions[usedAction].Inputs == nil || d.Actions[usedAction].Inputs[usedInput] == nil {
				validationErrors = append(validationErrors, as.formatError(action, step, "EA116", fmt.Sprintf("Input '%s' does not exist in local action '%s'", usedInput, uses), "action-step-uses-local-action-nonexisting-input"))
			}
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) validateUsesExternalAction(action string, step string, uses string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	err := d.DownloadExternalAction(uses)
	if err != nil {
		return validationErrors, err
	}
	if d.ExternalActions[uses] != nil {
		if d.ExternalActions[uses].Inputs != nil {
			for deaInputName, deaInput := range d.ExternalActions[uses].Inputs {
				if deaInput.Required {
					if as.With == nil || as.With[deaInputName] == "" {
						validationErrors = append(validationErrors, as.formatError(action, step, "EA115", fmt.Sprintf("Required input '%s' missing for external action '%s'", deaInputName, uses), "action-step-uses-external-action-missing-input"))
					}
				}
			}
		}
		if as.With != nil {
			for usedInput := range as.With {
				if d.ExternalActions[uses].Inputs == nil || d.ExternalActions[uses].Inputs[usedInput] == nil {
					validationErrors = append(validationErrors, as.formatError(action, step, "EA117", fmt.Sprintf("Input '%s' does not exist in external action '%s'", usedInput, uses), "action-step-uses-external-action-nonexisting-input"))
				}
			}
		}
	} else {
		validationErrors = append(validationErrors, as.formatError(action, step, "EA114", fmt.Sprintf("Call to non-existing external action '%s'", uses), "action-step-uses-nonexisting-external-action"))
	}

	return validationErrors, nil
}

func (as *ActionStep) validateCalledStepOutputs(action string, step string, uses string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	if as.Run == "" {
		return validationErrors, nil
	}
	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*steps\\.([a-zA-Z0-9\\-_]+)\\.outputs\\.([a-zA-Z0-9\\-_]+)[ ]*}}"))
	found := re.FindAllSubmatch([]byte(as.Run), -1)
	for _, f := range found {
		if d.Actions[action].Runs == nil {
			validationErrors = append(validationErrors, as.formatError(action, step, "EA118", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1])), "called-step-missing"))
			continue
		}

		found := d.Actions[action].Runs.IsStepOutputExist(string(f[1]), string(f[2]), d)
		if found == -1 {
			validationErrors = append(validationErrors, as.formatError(action, step, "EA118", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1])), "called-step-missing"))
		} else if found == -2 {
			validationErrors = append(validationErrors, as.formatError(action, step, "EA119", fmt.Sprintf("Called step with id '%s' output '%s' does not exist", string(f[1]), string(f[2])), "called-step-output-missing"))
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) formatError(action string, step string, code string, desc string, name string) string {
	return fmt.Sprintf("%s: %-60s %s (%s)", code, "action "+action+" step "+step, desc, name)
}
