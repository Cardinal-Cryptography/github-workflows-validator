package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"regexp"
	"strings"
	"io/ioutil"
)

type Workflow struct {
	Path     string
	Raw      []byte
	FileName string
	Name     string `yaml:"name"`
	Description string `yaml:"description"`
	Jobs map[string]*WorkflowJob `yaml:"jobs"`
}

func (w *Workflow) Init() error {
	pathSplit := strings.Split(w.Path, "/")
	w.FileName = pathSplit[len(pathSplit)-1]
	workflowName := strings.Replace(w.FileName, ".yaml", "", -1)
	w.Name = strings.Replace(workflowName, ".yml", "", -1)

	fmt.Fprintf(os.Stdout, "**** Reading %s ...\n", w.Path)
	b, err := ioutil.ReadFile(w.Path)
	if err != nil {
		return fmt.Errorf("Cannot read file %s: %w", w.Path, err)
	}
	w.Raw = b

	err = yaml.Unmarshal(w.Raw, &w)
	if err != nil {
		return fmt.Errorf("Cannot unmarshal file %s: %w", w.Path, err)
	}
	if w.Jobs != nil {
		for _, j := range w.Jobs {
			j.SetParentType("workflow")
		}
	}
	return nil
}

func (w *Workflow) Validate(d *DotGithub) ([]string, error) {
	var validationErrors []string
	verr, err := w.validateFileName()
	if err != nil {
		return validationErrors, err
	}
	if verr != "" {
		validationErrors = append(validationErrors, verr)
	}

	verrs, err := w.validateMissingFields()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = w.validateJobs(d)
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = w.validateCalledVarNames()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	return validationErrors, err
}

func (w *Workflow) formatError(code string, desc string, name string) string {
	return fmt.Sprintf("%s: %-40s %s (%s)", code, "workflow "+w.Name, desc, name)
}

func (w *Workflow) validateFileName() (string, error) {
	m, err := regexp.MatchString(`^[_]{0,1}[a-z0-9][a-z0-9\-]+\.y[a]{0,1}ml$`, w.FileName)
	if err != nil {
		return "", err
	}
	if !m {
		return w.formatError("EW101", "Workflow file name should contain alphanumeric characters and hyphens only", "workflow-filename-alphanumeric-and-hyphens"), nil
	}

	m, err = regexp.MatchString(`\.yml$`, w.Path)
	if err != nil {
		return "", err
	}
	if !m {
		return w.formatError("EW102", "Workflow file name should have .yml extension", "workflow-filename-yml-extension"), nil
	}
	return "", nil
}

func (w *Workflow) validateMissingFields() ([]string, error) {
	var validationErrors []string
	if w.Name == "" {
		validationErrors = append(validationErrors, w.formatError("EW103", "Workflow name is empty", "workflow-name-empty"))
	}
	if w.Description == "" {
		validationErrors = append(validationErrors, w.formatError("EW104", "Workflow description is empty", "workflow-description-empty"))
	}
	return validationErrors, nil
}

func (w *Workflow) validateJobs(d *DotGithub) ([]string, error) {
	var validationErrors []string
	if len(w.Jobs) == 1 {
		for jobName, _ := range w.Jobs {
			if jobName != "main" {
				validationErrors = append(validationErrors, w.formatError("EW106", "When workflow has only one job, it should be named 'main'", "workflow-only-job-not-main"))
			}
		}
	}

	for jobName, job := range w.Jobs {
		verrs, err := job.Validate(w.FileName, jobName, d)
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

func (w *Workflow) validateCalledVarNames() ([]string, error) {
	var validationErrors []string
	varTypes := []string{"env", "var", "secret"}
	for _, v := range varTypes {
		re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*%s\\.([a-zA-Z0-9\\-_]+)[ ]*}}", v))
		found := re.FindAllSubmatch(w.Raw, -1)
		for _, f := range found {
			m, err := regexp.MatchString(`^[A-Z][A-Z0-9_]+$`, string(f[1]))
			if err != nil {
				return validationErrors, err
			}
			if !m {
				validationErrors = append(validationErrors, w.formatError("EW107", fmt.Sprintf("Called variable name '%s' should contain uppercase alphanumeric characters and underscore only", string(f[1])), "called-variable-uppercase-alphanumeric-and-underscore"))
			}
		}
	}
	return validationErrors, nil
}
