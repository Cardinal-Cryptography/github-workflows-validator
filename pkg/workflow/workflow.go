package workflow

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Workflow struct {
	Path        string
	Raw         []byte
	FileName    string
	Name        string                  `yaml:"name"`
	Description string                  `yaml:"description"`
	Env         map[string]string       `yaml:"env"`
	Jobs        map[string]*WorkflowJob `yaml:"jobs"`
	On          *WorkflowOn             `yaml:"on"`
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

func (w *Workflow) Validate(d IDotGithub) ([]string, error) {
	var validationErrors []string
	verr, err := w.validateFileName()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErr(validationErrors, verr)

	verrs, err := w.validateEnv()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErrs(validationErrors, verrs)

	verrs, err = w.validateMissingFields()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErrs(validationErrors, verrs)

	verrs, err = w.validateOn()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErrs(validationErrors, verrs)

	verrs, err = w.validateJobs(d)
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErrs(validationErrors, verrs)

	verrs, err = w.validateCalledVarNames(d)
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErrs(validationErrors, verrs)

	verrs, err = w.validateCalledVarsNotInDoubleQuotes()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErrs(validationErrors, verrs)

	verrs, err = w.validateCalledInputs()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = w.appendErrs(validationErrors, verrs)

	return validationErrors, err
}

func (w *Workflow) appendErr(list []string, err string) []string {
	if err != "" {
		list = append(list, err)
	}
	return list
}

func (w *Workflow) appendErrs(list []string, errs []string) []string {
	if len(errs) > 0 {
		for _, err := range errs {
			list = w.appendErr(list, err)
		}
	}
	return list
}

func (w *Workflow) formatError(code string, desc string) string {
	return fmt.Sprintf("%s: %-80s %s", code, "workflow "+w.FileName, desc)
}

func (w *Workflow) validateFileName() (string, error) {
	m, err := regexp.MatchString(`^[_]{0,1}[a-z0-9][a-z0-9\-]+\.y[a]{0,1}ml$`, w.FileName)
	if err != nil {
		return "", err
	}
	if !m {
		return w.formatError("NW101", "Workflow file name should contain alphanumeric characters and hyphens only"), nil
	}

	m, err = regexp.MatchString(`\.yml$`, w.Path)
	if err != nil {
		return "", err
	}
	if !m {
		return w.formatError("NW102", "Workflow file name should have .yml extension"), nil
	}
	return "", nil
}

func (w *Workflow) validateEnv() ([]string, error) {
	var validationErrors []string
	if w.Env != nil {
		for envName := range w.Env {
			m, err := regexp.MatchString(`^[A-Z][A-Z0-9_]+$`, envName)
			if err != nil {
				return validationErrors, err
			}
			if !m {
				validationErrors = append(validationErrors, w.formatError("NW103", fmt.Sprintf("Env variable name '%s' should contain uppercase alphanumeric characters and underscore only", envName)))
			}
		}
	}
	return validationErrors, nil
}

func (w *Workflow) validateMissingFields() ([]string, error) {
	var validationErrors []string
	if w.Name == "" {
		validationErrors = append(validationErrors, w.formatError("NW104", "Workflow name is empty"))
	}
	return validationErrors, nil
}

func (w *Workflow) validateJobs(d IDotGithub) ([]string, error) {
	var validationErrors []string
	if len(w.Jobs) == 1 {
		for jobName := range w.Jobs {
			if jobName != "main" {
				validationErrors = append(validationErrors, w.formatError("NW106", "When workflow has only one job, it should be named 'main'"))
			}
		}
	}

	for jobName, job := range w.Jobs {
		verrs, err := job.Validate(w.FileName, jobName, d)
		if err != nil {
			return validationErrors, err
		}
		validationErrors = w.appendErrs(validationErrors, verrs)
		if job.Needs != nil {
			needsStr, ok := job.Needs.(string)
			if ok {
				if w.Jobs[needsStr] == nil {
					validationErrors = append(validationErrors, w.formatError("EW203", fmt.Sprintf("Job '%s' has invalid value '%s' in 'needs' field", jobName, needsStr)))
				}
			}

			needsList, ok := job.Needs.([]interface{})
			if ok {
				for _, neededJob := range needsList {
					if w.Jobs[neededJob.(string)] == nil {
						validationErrors = append(validationErrors, w.formatError("EW203", fmt.Sprintf("Job '%s' has invalid value '%s' in 'needs' field", jobName, neededJob.(string))))
					}
				}
			}
		}
	}
	return validationErrors, nil
}

func (w *Workflow) validateCalledVarNames(d IDotGithub) ([]string, error) {
	var validationErrors []string
	varTypes := []string{"env", "vars", "secrets"}
	for _, v := range varTypes {
		re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*%s\\.([a-zA-Z0-9\\-_]+)[ ]*}}", v))
		found := re.FindAllSubmatch(w.Raw, -1)
		for _, f := range found {
			m, err := regexp.MatchString(`^[A-Z][A-Z0-9_]+$`, string(f[1]))
			if err != nil {
				return validationErrors, err
			}
			if !m {
				validationErrors = append(validationErrors, w.formatError("NW107", fmt.Sprintf("Called variable name '%s' should contain uppercase alphanumeric characters and underscore only", string(f[1]))))
			}

			if v == "vars" && d.IsVarsFileExist() && !d.IsVarExist(string(f[1])) {
				validationErrors = append(validationErrors, w.formatError("EW254", fmt.Sprintf("Called variable '%s' does not exist in provided list of available vars", string(f[1]))))
			}

			if v == "secrets" && d.IsSecretsFileExist() && !d.IsSecretExist(string(f[1])) {
				validationErrors = append(validationErrors, w.formatError("EW255", fmt.Sprintf("Called secret '%s' does not exist in provided list of available secrets", string(f[1]))))
			}
		}
	}

	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*([a-zA-Z0-9\\-_]+)[ ]*}}"))
	found := re.FindAllSubmatch(w.Raw, -1)
	for _, f := range found {
		if string(f[1]) != "false" && string(f[1]) != "true" {
			validationErrors = append(validationErrors, w.formatError("EW201", fmt.Sprintf("Called variable '%s' is invalid", string(f[1]))))
		}
	}
	return validationErrors, nil
}

func (w *Workflow) validateOn() ([]string, error) {
	var validationErrors []string
	if w.On != nil {
		verrs, err := w.On.Validate(w.FileName)
		if err != nil {
			return validationErrors, err
		}
		validationErrors = w.appendErrs(validationErrors, verrs)
	}
	return validationErrors, nil
}

func (w *Workflow) validateCalledInputs() ([]string, error) {
	var validationErrors []string
	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*inputs\\.([a-zA-Z0-9\\-_]+)[ ]*}}"))
	found := re.FindAllSubmatch(w.Raw, -1)
	for _, f := range found {
		notInInputs := true
		if w.On != nil {
			if w.On.WorkflowCall != nil && w.On.WorkflowCall.Inputs != nil && w.On.WorkflowCall.Inputs[string(f[1])] != nil {
				notInInputs = false
			}
			if w.On.WorkflowDispatch != nil && w.On.WorkflowDispatch.Inputs != nil && w.On.WorkflowDispatch.Inputs[string(f[1])] != nil {
				notInInputs = false
			}
		}
		if notInInputs {
			validationErrors = append(validationErrors, w.formatError("EW202", fmt.Sprintf("Called input '%s' does not exist", string(f[1]))))
		}
	}
	return validationErrors, nil
}

func (w *Workflow) validateCalledVarsNotInDoubleQuotes() ([]string, error) {

	var validationErrors []string
	re := regexp.MustCompile(`\"\${{[ ]*([a-zA-Z0-9\\-_.]+)[ ]*}}\"`)
	found := re.FindAllSubmatch(w.Raw, -1)
	for _, f := range found {
		validationErrors = append(validationErrors, w.formatError("WW201", fmt.Sprintf("Called variable '%s' may not need to be in double quotes", string(f[1]))))
	}
	return validationErrors, nil
}
