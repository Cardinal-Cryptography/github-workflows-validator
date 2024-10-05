package action

import (
	"fmt"
	"regexp"
	"strings"
)

type ActionStep struct {
	ParentType string
	Name       string            `yaml:"name"`
	Id         string            `yaml:"id"`
	Uses       string            `yaml:"uses"`
	Shell      string            `yaml:"bash"`
	Env        map[string]string `yaml:"env"`
	Run        string            `yaml:"run"`
	With       map[string]string `yaml:"with"`
}

func (as *ActionStep) Validate(action string, workflowJob string, name string, d IDotGithub) ([]string, error) {
	var validationErrors []string

	verrs, err := as.validateUses(action, workflowJob, name, as.Uses, d)
	if err != nil {
		return validationErrors, err
	}
	validationErrors = as.appendErrs(validationErrors, verrs)

	verrs, err = as.validateEnv(action, workflowJob, name)
	if err != nil {
		return validationErrors, err
	}
	validationErrors = as.appendErrs(validationErrors, verrs)

	verrs, err = as.validateCalledStepOutputs(action, workflowJob, name, as.Uses, d)
	if err != nil {
		return validationErrors, err
	}
	validationErrors = as.appendErrs(validationErrors, verrs)

	verrs, err = as.validateCalledEnv(action, workflowJob, name, as.Uses, d)
	if err != nil {
		return validationErrors, err
	}
	validationErrors = as.appendErrs(validationErrors, verrs)

	return validationErrors, nil
}

func (as *ActionStep) appendErr(list []string, err string) []string {
	if err != "" {
		list = append(list, err)
	}
	return list
}

func (as *ActionStep) appendErrs(list []string, errs []string) []string {
	if len(errs) > 0 {
		for _, err := range errs {
			list = as.appendErr(list, err)
		}
	}
	return list
}

func (as *ActionStep) validateUses(action string, workflowJob string, name string, uses string, d IDotGithub) ([]string, error) {
	var validationErrors []string
	if uses == "" {
		return validationErrors, nil
	}

	if strings.HasPrefix(as.Uses, "./.github/") {
		verrs, err := as.validateUsesLocalAction(action, workflowJob, name, as.Uses, d)
		if err != nil {
			return validationErrors, err
		}
		validationErrors = as.appendErrs(validationErrors, verrs)
	} else {
		m, err := regexp.MatchString(`[a-zA-Z0-9\-\_]+\/[a-zA-Z0-9\-\_]+(\/[a-zA-Z0-9\-\_]){0,1}@[a-zA-Z0-9\.\-\_]+`, as.Uses)
		if err != nil {
			return validationErrors, err
		}
		if !m {
			if as.ParentType == "workflow" {
				validationErrors = append(validationErrors, as.formatErrorForWorkflow(action, workflowJob, name, "EW801", fmt.Sprintf("Path to external action '%s' is invalid", as.Uses)))
			} else {
				validationErrors = append(validationErrors, as.formatError(action, name, "EA801", fmt.Sprintf("Path to external action '%s' is invalid", as.Uses)))
			}
		} else {
			verrs, err := as.validateUsesExternalAction(action, workflowJob, name, as.Uses, d)
			if err != nil {
				return validationErrors, err
			}
			validationErrors = as.appendErrs(validationErrors, verrs)
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) validateUsesLocalAction(actionName string, workflowJobName string, step string, uses string, d IDotGithub) ([]string, error) {
	var validationErrors []string
	m, err := regexp.MatchString(`^\.\/\.github\/actions\/[a-z0-9\-]+$`, uses)
	if err != nil {
		return validationErrors, err
	}
	if !m {
		if as.ParentType == "workflow" {
			validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW802", fmt.Sprintf("Path to local action '%s' is invalid", uses)))
		} else {
			validationErrors = append(validationErrors, as.formatError(actionName, step, "EA802", fmt.Sprintf("Path to local action '%s' is invalid", uses)))
		}
	}

	action := d.GetAction(strings.Replace(uses, "./.github/actions/", "", -1))
	if action == nil {
		if as.ParentType == "workflow" {
			validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW803", fmt.Sprintf("Call to non-existing local action '%s'", uses)))
		} else {
			validationErrors = append(validationErrors, as.formatError(actionName, step, "EA803", fmt.Sprintf("Call to non-existing local action '%s'", uses)))
		}
		return validationErrors, nil
	}

	if action.Inputs != nil {
		for daInputName, daInput := range action.Inputs {
			if daInput.Required {
				if as.With == nil || as.With[daInputName] == "" {
					if as.ParentType == "workflow" {
						validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW804", fmt.Sprintf("Required input '%s' missing for local action '%s'", daInputName, uses)))
					} else {
						validationErrors = append(validationErrors, as.formatError(actionName, step, "EA804", fmt.Sprintf("Required input '%s' missing for local action '%s'", daInputName, uses)))
					}
				}
			}
		}
	}
	if as.With != nil {
		for usedInput := range as.With {
			if action.Inputs == nil || action.Inputs[usedInput] == nil {
				if as.ParentType == "workflow" {
					validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW805", fmt.Sprintf("Input '%s' does not exist in local action '%s'", usedInput, uses)))
				} else {
					validationErrors = append(validationErrors, as.formatError(actionName, step, "EA805", fmt.Sprintf("Input '%s' does not exist in local action '%s'", usedInput, uses)))
				}
			}
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) validateUsesExternalAction(actionName string, workflowJobName string, step string, uses string, d IDotGithub) ([]string, error) {
	var validationErrors []string
	err := d.DownloadExternalAction(uses)
	if err != nil {
		return validationErrors, err
	}

	action := d.GetExternalAction(uses)
	if action != nil {
		if action.Inputs != nil {
			for deaInputName, deaInput := range action.Inputs {
				if deaInput.Required {
					if as.With == nil || as.With[deaInputName] == "" {
						if as.ParentType == "workflow" {
							validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW806", fmt.Sprintf("Required input '%s' missing for external action '%s'", deaInputName, uses)))
						} else {
							validationErrors = append(validationErrors, as.formatError(actionName, step, "EA806", fmt.Sprintf("Required input '%s' missing for external action '%s'", deaInputName, uses)))
						}
					}
				}
			}
		}
		if as.With != nil {
			for usedInput := range as.With {
				if action.Inputs == nil || action.Inputs[usedInput] == nil {
					if as.ParentType == "workflow" {
						validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW807", fmt.Sprintf("Input '%s' does not exist in external action '%s'", usedInput, uses)))
					} else {
						validationErrors = append(validationErrors, as.formatError(actionName, step, "EA807", fmt.Sprintf("Input '%s' does not exist in external action '%s'", usedInput, uses)))
					}
				}
			}
		}
	} else {
		if as.ParentType == "workflow" {
			validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW808", fmt.Sprintf("Call to non-existing external action '%s'", uses)))
		} else {
			validationErrors = append(validationErrors, as.formatError(actionName, step, "EA808", fmt.Sprintf("Call to non-existing external action '%s'", uses)))
		}
	}

	return validationErrors, nil
}

func (as *ActionStep) validateEnv(action string, workflowJob string, step string) ([]string, error) {
	var validationErrors []string
	if as.Env != nil {
		for envName := range as.Env {
			m, err := regexp.MatchString(`^[A-Z][A-Z0-9_]+$`, envName)
			if err != nil {
				return validationErrors, err
			}
			if !m {
				if as.ParentType == "workflow" {
					validationErrors = append(validationErrors, as.formatErrorForWorkflow(action, workflowJob, step, "NW701", fmt.Sprintf("Env variable name '%s' should contain uppercase alphanumeric characters and underscore only", envName)))
				} else {
					validationErrors = append(validationErrors, as.formatError(action, step, "NA701", fmt.Sprintf("Env variable name '%s' should contain uppercase alphanumeric characters and underscore only", envName)))
				}
			}
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) validateCalledStepOutputs(actionName string, workflowJobName string, step string, uses string, d IDotGithub) ([]string, error) {
	var validationErrors []string
	if as.Run == "" {
		return validationErrors, nil
	}
	runAndEnvsStr := as.Run
	if as.Env != nil {
		for _, envVal := range as.Env {
			runAndEnvsStr = runAndEnvsStr + " " + envVal
		}
	}
	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*steps\\.([a-zA-Z0-9\\-_]+)\\.outputs\\.([a-zA-Z0-9\\-_]+)[ ]*}}"))
	found := re.FindAllSubmatch([]byte(runAndEnvsStr), -1)
	for _, f := range found {
		if as.ParentType == "workflow" {
			if !d.IsWorkflowJobStepOutputExist(actionName, workflowJobName, string(f[1]), string(f[2])) {
				validationErrors = append(validationErrors, as.formatErrorForWorkflow(actionName, workflowJobName, step, "EW811", fmt.Sprintf("Called step with id '%s' output '%s' does not exist", string(f[1]))))
				continue
			}
		} else {
			action := d.GetAction(actionName)
			if action.Runs == nil {
				validationErrors = append(validationErrors, as.formatError(actionName, step, "EA809", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1]))))
				continue
			}

			found := action.Runs.IsStepOutputExist(string(f[1]), string(f[2]), d)
			if found == -1 {
				validationErrors = append(validationErrors, as.formatError(actionName, step, "EA809", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1]))))
			} else if found == -2 {
				validationErrors = append(validationErrors, as.formatError(actionName, step, "EA811", fmt.Sprintf("Called step with id '%s' output '%s' does not exist", string(f[1]), string(f[2]))))
			}
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) validateCalledEnv(action string, workflowJob string, step string, uses string, d IDotGithub) ([]string, error) {
	var validationErrors []string
	if as.Run == "" {
		return validationErrors, nil
	}
	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*env\\.([a-zA-Z0-9\\-_]+)[ ]*}}"))
	found := re.FindAllSubmatch([]byte(as.Run), -1)
	for _, f := range found {
		if !strings.HasPrefix(string(f[1]), "GITHUB_") && !strings.HasPrefix(string(f[1]), "RUNNER_") && string(f[1]) != "CI" && as.ParentType == "workflow" {
			found := false
			if as.Env != nil && as.Env[string(f[1])] != "" {
				found = true
			}
			if !found && d.IsEnvExistInWorkflowOrItsJob(action, workflowJob, string(f[1])) {
				found = true
			}
			if !found {
				validationErrors = append(validationErrors, as.formatErrorForWorkflow(action, workflowJob, step, "WW101", fmt.Sprintf("Called env var '%s' not found in global, job or step 'env' block - check it", string(f[1]))))
			}
		}
	}
	return validationErrors, nil
}

func (as *ActionStep) formatError(action string, step string, code string, desc string) string {
	return fmt.Sprintf("%s: %-60s %s", code, "action "+action+" step "+step, desc)
}

func (as *ActionStep) formatErrorForWorkflow(workflow string, workflowJob string, step string, code string, desc string) string {
	return fmt.Sprintf("%s: %-80s %s", code, "workflow "+workflow+" job "+workflowJob+" step "+step, desc)
}
