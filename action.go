package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
)

type Action struct {
	Path        string
	Raw         []byte
	DirName     string
	Name        string                   `yaml:"name"`
	Description string                   `yaml:"description"`
	Inputs      map[string]*ActionInput  `yaml:"inputs"`
	Outputs     map[string]*ActionOutput `yaml:"outputs"`
	Runs        *ActionRuns              `yaml:"runs"`
}

func (a *Action) Init(fromRaw bool) error {
	if !fromRaw {
		fmt.Fprintf(os.Stdout, "**** Reading %s ...\n", a.Path)
		b, err := ioutil.ReadFile(a.Path)
		if err != nil {
			return fmt.Errorf("Cannot read file %s: %w", a.Path, err)
		}
		a.Raw = b
	}
	err := yaml.Unmarshal(a.Raw, &a)
	if err != nil {
		return fmt.Errorf("Cannot unmarshal file %s: %w", a.Path, err)
	}
	if a.Runs != nil {
		a.Runs.SetParentType("action")
	}
	return nil
}

func (a *Action) Validate(d *DotGithub) ([]string, error) {
	var validationErrors []string
	verr, err := a.validateDirName()
	if err != nil {
		return validationErrors, err
	}
	if verr != "" {
		validationErrors = append(validationErrors, verr)
	}

	verr, err = a.validateFileName()
	if err != nil {
		return validationErrors, err
	}
	if verr != "" {
		validationErrors = append(validationErrors, verr)
	}

	verrs, err := a.validateMissingFields()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = a.validateInputs()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = a.validateOutputs()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = a.validateCalledVarNames()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = a.validateCalledInputs()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = a.validateCalledStepOutputs()
	if err != nil {
		return validationErrors, err
	}
	if len(verrs) > 0 {
		for _, verr := range verrs {
			validationErrors = append(validationErrors, verr)
		}
	}

	verrs, err = a.validateSteps(d)
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

func (a *Action) formatError(code string, desc string, name string) string {
	return fmt.Sprintf("%s: %-40s %s (%s)", code, "action "+a.DirName, desc, name)
}

func (a *Action) validateDirName() (string, error) {
	m, err := regexp.MatchString(`^[a-z0-9][a-z0-9\-]+$`, a.DirName)
	if err != nil {
		return "", err
	}
	if !m {
		return a.formatError("EA101", "Action directory name should contain lowercase alphanumeric characters and hyphens only", "action-dirname-lowercase-alphanumeric-and-hyphens"), nil
	}
	return "", nil
}

func (a *Action) validateFileName() (string, error) {
	m, err := regexp.MatchString(`\.yml$`, a.Path)
	if err != nil {
		return "", err
	}
	if !m {
		return a.formatError("EA102", "Action file name should have .yml extension", "action-filename-yml-extension"), nil
	}
	return "", nil
}

func (a *Action) validateMissingFields() ([]string, error) {
	var validationErrors []string
	if a.Name == "" {
		validationErrors = append(validationErrors, a.formatError("EA103", "Action name is empty", "action-name-empty"))
	}
	if a.Description == "" {
		validationErrors = append(validationErrors, a.formatError("EA104", "Action description is empty", "action-description-empty"))
	}
	return validationErrors, nil
}

func (a *Action) validateInputs() ([]string, error) {
	var validationErrors []string
	if a.Inputs != nil {
		for inputName, input := range a.Inputs {
			verrs, err := input.Validate(a.DirName, inputName)
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

func (a *Action) validateOutputs() ([]string, error) {
	var validationErrors []string
	if a.Outputs != nil {
		for outputName, output := range a.Outputs {
			verrs, err := output.Validate(a.DirName, outputName)
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

func (a *Action) validateCalledVarNames() ([]string, error) {
	var validationErrors []string
	varTypes := []string{"env", "var", "secret"}
	for _, v := range varTypes {
		re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*%s\\.([a-zA-Z0-9\\-_]+)[ ]*}}", v))
		found := re.FindAllSubmatch(a.Raw, -1)
		for _, f := range found {
			m, err := regexp.MatchString(`^[A-Z][A-Z0-9_]+$`, string(f[1]))
			if err != nil {
				return validationErrors, err
			}
			if !m {
				validationErrors = append(validationErrors, a.formatError("EA109", fmt.Sprintf("Called variable name '%s' should contain uppercase alphanumeric characters and underscore only", string(f[1])), "called-variable-uppercase-alphanumeric-and-underscore"))
			}
		}
	}
	return validationErrors, nil
}

func (a *Action) validateCalledInputs() ([]string, error) {
	var validationErrors []string
	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*inputs\\.([a-zA-Z0-9\\-_]+)[ ]*}}"))
	found := re.FindAllSubmatch(a.Raw, -1)
	for _, f := range found {
		if a.Inputs == nil || a.Inputs[string(f[1])] == nil {
			validationErrors = append(validationErrors, a.formatError("EA110", fmt.Sprintf("Called input '%s' does not exist", string(f[1])), "called-input-missing"))
		}
	}
	return validationErrors, nil
}

func (a *Action) validateCalledStepOutputs() ([]string, error) {
	var validationErrors []string
	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*steps\\.([a-zA-Z0-9\\-_]+)\\.outputs\\.[a-zA-Z0-9\\-_]+[ ]*}}"))
	found := re.FindAllSubmatch(a.Raw, -1)
	for _, f := range found {
		if a.Runs == nil {
			validationErrors = append(validationErrors, a.formatError("EA118", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1])), "called-step-missing"))
		} else {
			if !a.Runs.IsStepExist(string(f[1])) {
				validationErrors = append(validationErrors, a.formatError("EA118", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1])), "called-step-missing"))
			}
		}
	}
	return validationErrors, nil
}

func (a *Action) validateSteps(d *DotGithub) ([]string, error) {
	var validationErrors []string
	if a.Runs != nil {
		verrs, err := a.Runs.Validate(a.DirName, d)
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
