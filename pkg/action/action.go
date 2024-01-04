package action

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"regexp"
	"strings"
	"net/http"
)

type Action struct {
	path        string
	raw         []byte
	dirName     string
	Name        string                   `yaml:"name"`
	Description string                   `yaml:"description"`
	Inputs      map[string]*ActionInput  `yaml:"inputs"`
	Outputs     map[string]*ActionOutput `yaml:"outputs"`
	Runs        *ActionRuns              `yaml:"runs"`
}

// NewFromFile returns pointer to Action instance and parses out specified path to YAML file.
func NewFromFile(dirName string, path string) (*Action, error) {
	fmt.Fprintf(os.Stdout, "**** Reading %s ...\n", s.path)
	b, err := ioutil.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read file %s: %w", s.path, err)
	}

	s, err := NewFromBytes(dirName, b)
	if err != nil {
		return nil, fmt.Errorf("Cannot create from file %s: %w", s.path, err)
	}
	s.path = path

	return s, nil
}

// NewFromBytes returns pointer to Action instance and parses out YAML data specified as bytes.
func NewFromBytes(dirName string, b []byte) (*Action, error) {
	s := &Action{
		raw: b,
		dirName: dirName,
	}

	err := yaml.Unmarshal(s.raw, a)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal action data: %w", err)
	}
	if s.Runs != nil {
		s.Runs.SetParentType("action")
	}

	return s, nil
}

// NewFromExternal returns pointer to Action instance from a specified external path, eg. organization/repo@v3.  YAML
// file of such action is downloaded and unmarshalled into struct.
func NewFromExternal(path string) (*Action, error) {
	repoVersion := strings.Split(path, "@")
	ownerRepoDir := strings.SplitN(repoVersion[0], "/", 3)
	directory := ""
	if len(ownerRepoDir) > 2 {
		directory = "/" + ownerRepoDir[2]
	}
	actionURLPrefix := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s", ownerRepoDir[0], ownerRepoDir[1], repoVersion[1])

	req, err := http.NewRequest("GET", actionURLPrefix+directory+"/action.yml", strings.NewReader(""))
	if err != nil {
		return err
	}
	c := &http.Client{}
	resp, err := c.Do(req)

	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		req, err = http.NewRequest("GET", actionURLPrefix+directory+"/action.yaml", strings.NewReader(""))
		if err != nil {
			return err
		}
		resp, err = c.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return nil
		}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	s, err := NewFromBytes(dirName, b)
	if err != nil {
		return nil, fmt.Errorf("Cannot create from file %s: %w", s.path, err)
	}
	s.path = path

	return s, nil
}


func (a *Action) Validate(d IDotGithub) ([]string, error) {
	var validationErrors []string

	verr, err := a.validateDirName()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErr(validationErrors, verr)

	verr, err = a.validateFileName()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErr(validationErrors, verr)

	verrs, err := a.validateMissingFields()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	verrs, err = a.validateInputs()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	verrs, err = a.validateOutputs()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	verrs, err = a.validateCalledVarNames()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	verrs, err = a.validateCalledInputs()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	verrs, err = a.validateCalledStepOutputs()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	verrs, err = a.validateCalledVarsNotInDoubleQuotes()
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	verrs, err = a.validateSteps(d)
	if err != nil {
		return validationErrors, err
	}
	validationErrors = a.appendErrs(validationErrors, verrs)

	return validationErrors, err
}

func (a *Action) appendErr(list []string, err string) []string {
	if err != "" {
		list = append(list, err)
	}
	return list
}

func (a *Action) appendErrs(list []string, errs []string) []string {
	if len(errs) > 0 {
		for _, err := range errs {
			list = a.appendErr(list, err)
		}
	}
	return list
}

func (a *Action) formatError(code string, desc string) string {
	return fmt.Sprintf("%s: %-40s %s", code, "action "+a.DirName, desc)
}

func (a *Action) validateDirName() (string, error) {
	m, err := regexp.MatchString(`^[a-z0-9][a-z0-9\-]+$`, a.DirName)
	if err != nil {
		return "", err
	}
	if !m {
		return a.formatError("NA101", "Action directory name should contain lowercase alphanumeric characters and hyphens only"), nil
	}
	return "", nil
}

func (a *Action) validateFileName() (string, error) {
	m, err := regexp.MatchString(`\.yml$`, a.Path)
	if err != nil {
		return "", err
	}
	if !m {
		return a.formatError("NA102", "Action file name should have .yml extension"), nil
	}
	return "", nil
}

func (a *Action) validateMissingFields() ([]string, error) {
	var validationErrors []string
	if a.Name == "" {
		validationErrors = append(validationErrors, a.formatError("NA103", "Action name is empty"))
	}
	if a.Description == "" {
		validationErrors = append(validationErrors, a.formatError("NA104", "Action description is empty"))
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
			validationErrors = a.appendErrs(validationErrors, verrs)
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
			validationErrors = a.appendErrs(validationErrors, verrs)
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
				validationErrors = append(validationErrors, a.formatError("NA105", fmt.Sprintf("Called variable name '%s' should contain uppercase alphanumeric characters and underscore only", string(f[1]))))
			}
		}
	}

	re := regexp.MustCompile(fmt.Sprintf("\\${{[ ]*([a-zA-Z0-9\\-_]+)[ ]*}}"))
	found := re.FindAllSubmatch(a.Raw, -1)
	for _, f := range found {
		if string(f[1]) != "false" && string(f[1]) != "true" {
			validationErrors = append(validationErrors, a.formatError("EA201", fmt.Sprintf("Called variable '%s' is invalid", string(f[1]))))
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
			validationErrors = append(validationErrors, a.formatError("EA202", fmt.Sprintf("Called input '%s' does not exist", string(f[1]))))
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
			validationErrors = append(validationErrors, a.formatError("EA203", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1]))))
		} else {
			if !a.Runs.IsStepExist(string(f[1])) {
				validationErrors = append(validationErrors, a.formatError("EA204", fmt.Sprintf("Called step with id '%s' does not exist", string(f[1]))))
			}
		}
	}
	return validationErrors, nil
}

func (a *Action) validateCalledVarsNotInDoubleQuotes() ([]string, error) {
	var validationErrors []string
	re := regexp.MustCompile(`\"\${{[ ]*([a-zA-Z0-9\\-_.]+)[ ]*}}\"`)
	found := re.FindAllSubmatch(a.Raw, -1)
	for _, f := range found {
		validationErrors = append(validationErrors, a.formatError("WW201", fmt.Sprintf("Called variable '%s' may not need to be in double quotes", string(f[1]))))
	}
	return validationErrors, nil
}

func (a *Action) validateSteps(d IDotGithub) ([]string, error) {
	var validationErrors []string
	if a.Runs != nil {
		verrs, err := a.Runs.Validate(a.DirName, d)
		if err != nil {
			return validationErrors, err
		}
		validationErrors = a.appendErrs(validationErrors, verrs)
	}
	return validationErrors, nil
}
