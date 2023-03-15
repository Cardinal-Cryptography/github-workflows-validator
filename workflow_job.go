package main

import (
	"regexp"
	"fmt"
	"strconv"
	"strings"
)

type WorkflowJob struct {
	Name string `yaml:"name"`
	Uses string `yaml:"uses"`
	RunsOn string `yaml:"runs-on"`
	Steps []*ActionStep `yaml:"steps"`
	Env map[string]string `yaml:"env"`
}

func (wj *WorkflowJob) SetParentType(t string) {
	for _, s := range wj.Steps {
		s.ParentType = t
	}
}

func (wj *WorkflowJob) Validate(workflow string, job string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	verr, err := wj.validateName(workflow, job, d)
	if err != nil {
		return validationErrors, err
	}
	if verr != "" {
		validationErrors = append(validationErrors, verr)
	}

	if wj.Uses == "" && wj.RunsOn == "" {
		validationErrors = append(validationErrors, wj.formatError(workflow, job, "EW120", "Workflow job name should have either 'uses' or 'runs-on'", "workflow-job-uses-or-runs-on-not-found"))
	}
	if strings.Contains(wj.RunsOn, "latest") {
		validationErrors = append(validationErrors, wj.formatError(workflow, job, "EW120", "Workflow job should not have 'latest' in 'runs-on'", "workflow-job-latest-in-runs-on"))
	}

	verrs, err := wj.validateSteps(workflow, job, d)
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

func (wj *WorkflowJob) validateName(workflow string, job string, d *DotGithub) (string, error) {
	m, err := regexp.MatchString(`^[a-z0-9][a-z0-9\-]+$`, job)
	if err != nil {
		return "", err
	}
	if !m {
		return wj.formatError(workflow, job, "EW105", "Workflow job name should contain lowercase alphanumeric characters and hyphens only", "workflow-job-lowercase-alphanumeric-and-hyphens"), nil
	}
	return "", nil
}

func (wj *WorkflowJob) formatError(workflow string, job string, code string, desc string, name string) string {
	return fmt.Sprintf("%s: %-60s %s (%s)", code, "workflow "+workflow+" job "+job, desc, name)
}

func (wj *WorkflowJob) IsStepExist(id string) bool {
	for _, s := range wj.Steps {
		if s.Id == id {
			return true
		}
	}
	return false
}

func (wj *WorkflowJob) validateSteps(workflow string, job string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	if wj.Steps != nil {
		for i, s := range wj.Steps {
			verrs, err := s.Validate(workflow, job, strconv.Itoa(i), d)
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

func (wj *WorkflowJob) IsStepOutputExist(step string, output string, d *DotGithub) int {
	for _, s := range wj.Steps {
		if s.Id != step {
			continue
		}

		if s.Uses == "" && s.Run != "" {
			re := regexp.MustCompile(`echo[ ]+"([a-zA-Z0-9\-_]+)=.*"[ ]+.*>>[ ]+\$GITHUB_OUTPUT`)
			found := re.FindAllSubmatch([]byte(s.Run), -1)
			for _, f := range found {
				if output == string(f[1]) {
					return 0
				}
			}
			return -2
		}

		re := regexp.MustCompile(`^\.\/\.github\/actions\/[a-z0-9\-]+$`)
		m := re.MatchString(s.Uses)
		if m {
			usedAction := strings.Replace(s.Uses, "./.github/actions/", "", -1)
			if d.Actions != nil || d.Actions[usedAction] != nil {
				for duaOutputName, _ := range d.Actions[usedAction].Outputs {
					if duaOutputName == output {
						return 0
					}
				}
			}
		}

		re = regexp.MustCompile(`[a-zA-Z0-9\-\_]+\/[a-zA-Z0-9\-\_]+@[a-zA-Z0-9\.\-\_]+`)
		m = re.MatchString(s.Uses)
		if m {
			if d.Actions != nil || d.ExternalActions[s.Uses] != nil {
				for duaOutputName, _ := range d.ExternalActions[s.Uses].Outputs {
					if duaOutputName == output {
						return 0
					}
				}
			}
		}

		return -2
	}
	return -1
}
