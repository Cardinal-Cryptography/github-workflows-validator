package main

import (
	"fmt"
	_ "gopkg.in/yaml.v2"
	"os"
	"strings"
	"regexp"
)

type Workflow struct {
	Path string
	FileName string
	Name string
}

func (w *Workflow) Init() error {
	pathSplit := strings.Split(w.Path, "/")
	w.FileName = pathSplit[len(pathSplit)-1]
	workflowName := strings.Replace(w.FileName, ".yaml", "", -1)
	w.Name = strings.Replace(workflowName, ".yml", "", -1)

	fmt.Fprintf(os.Stdout, "**** Reading %s ...\n", w.Path)
	return nil
}

func (w *Workflow) Validate() ([]string, error) {
	var validationErrors []string
	verr, err := w.validateFileName()
	if err != nil {
		return validationErrors, err
	}
	if verr != "" {
		validationErrors = append(validationErrors, verr)
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
