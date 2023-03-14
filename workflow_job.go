package main

import (
	"regexp"
	"fmt"
)

type WorkflowJob struct {
	Name string `yaml:"name"`
	RunsOn string `yaml:"runs-on"`
	Steps []*ActionStep `yaml:"steps"`
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
