package dotgithub

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Cardinal-Cryptography/github-actions-validator/pkg/action"
	"github.com/Cardinal-Cryptography/github-actions-validator/pkg/workflow"
)

// DotGitHub relates to .github directory contents, hence, it consists of actions and workflows.
type DotGithub struct {
	path            string
	actions         map[string]*action.Action
	externalActions map[string]*action.Action
	workflows       map[string]*workflow.Workflow

	VarsFile        string
	SecretsFile     string
	Vars            map[string]bool
	Secrets         map[string]bool
	
}

// NewFromPath return pointer to new DotGitHub instance with specified path, which is scanned for workflow and action 
// YAML files.  In case of any problem, log.Fatal is called.
func NewFromPath(path string) *DotGitHub {
	s := &DotGithub{
		path:        path,
		actions:     map[string]*action.Action{},
		externalActions: map[string]*action.Action{},
		workflows: map[string]*workflow.Workflow{},
	}

	s.getActions()
	s.getWorkflows()
	s.getVars()
	s.getSecrets()

	return s
}

// DownloadExternalAction downloads an external action, specified by a path like organization/repo@v3, gets its YAML
// and parses it out and caches in the DotGitHub instance.
func (d *DotGithub) DownloadExternalAction(path string) error {
	if d.ExternalActions[path] != nil {
		return nil
	}

	newAction, err := action.NewFromExternal(path)
	if err != nil {
		return err
	}
	d.ExternalActions[path] = newAction
	return nil
}

// getActions scans sub-directories in "actions" directory and parses out any of them containing either action.yml or
// action.yaml.
func (d *DotGithub) getActions() {
	actionsPath := filepath.Join(d.Path, "actions")
	entries, err := os.ReadDir(actionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatal(err)
	}
	for _, e := range entries {
		entryPath := filepath.Join(actionsPath, e.Name())
		fileInfo, err := os.Stat(entryPath)
		if err != nil {
			log.Fatal(err)
		}
		if !fileInfo.IsDir() {
			continue
		}

		actionYMLPath := filepath.Join(entryPath, "action.yml")
		_, err = os.Stat(actionYMLPath)
		ymlNotFound := os.IsNotExist(err)
		if err != nil && !ymlNotFound {
			log.Fatal(err)
		}
		if ymlNotFound {
			actionYAMLPath := filepath.Join(entryPath, "action.yaml")
			_, err = os.Stat(actionYAMLPath)
			yamlNotFound := os.IsNotExist(err)
			if err != nil && !yamlNotFound {
				log.Fatal(err)
			}
			if !yamlNotFound {
				actionYMLPath = actionYAMLPath
			} else {
				continue
			}
		}

		newAction, err := action.NewFromFile(e.Name(), actionYMLPath)
		if err != nil {
			log.Fatal(err)
		}
		d.Actions[e.Name()] = newAction
	}
}

// getWorkflows scans "workflows" directory and parses out any .yml and .yaml files
func (d *DotGithub) getWorkflows() {
	workflowsPath := filepath.Join(d.Path, "workflows")
	entries, err := os.ReadDir(workflowsPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		m, err := regexp.MatchString("\\.y[a]{0,1}ml$", e.Name())
		if err != nil {
			log.Fatal(err)
		}
		if !m {
			continue
		}

		entryPath := filepath.Join(workflowsPath, e.Name())
		fileInfo, err := os.Stat(entryPath)
		if err != nil {
			log.Fatal(err)
		}
		if !fileInfo.Mode().IsRegular() {
			continue
		}

		newWorkflow, err := workflow.NewFromFile(e.Name(), entryPath)
		if err != nil {
			log.Fatal(err)
		}
		d.Workflows[e.Name()] = newWorkflow
	}
}






func (d *DotGithub) getVars() {
	d.Vars = make(map[string]bool)
	if d.VarsFile != "" {
		fmt.Fprintf(os.Stdout, "**** Reading file with list of possible variable names %s ...\n", d.VarsFile)
		b, err := ioutil.ReadFile(d.VarsFile)
		if err != nil {
			log.Fatal(fmt.Errorf("Cannot read file %s: %w", d.VarsFile, err))
		}
		l := strings.Fields(string(b))
		for _, v := range l {
			d.Vars[v] = true
		}
	}
}

func (d *DotGithub) getSecrets() {
	d.Secrets = make(map[string]bool)
	if d.SecretsFile != "" {
		fmt.Fprintf(os.Stdout, "**** Reading file with list of possible secret names %s ...\n", d.SecretsFile)
		b, err := ioutil.ReadFile(d.SecretsFile)
		if err != nil {
			log.Fatal(fmt.Errorf("Cannot read file %s: %w", d.SecretsFile, err))
		}
		l := strings.Fields(string(b))
		for _, s := range l {
			d.Secrets[s] = true
		}
	}
}

func (d *DotGithub) validateActions() ([]string, error) {
	var validationErrors []string
	for _, a := range d.Actions {
		verrs, err := a.Validate(d)
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

func (d *DotGithub) validateWorkflows() ([]string, error) {
	var validationErrors []string
	for _, w := range d.Workflows {
		verrs, err := w.Validate(d)
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

func (d *DotGithub) Validate() ([]string, error) {
	var validationErrors []string

	verrs, err := d.validateActions()
	if err != nil {
		return validationErrors, err
	}
	for _, v := range verrs {
		validationErrors = append(validationErrors, v)
	}

	verrs, err = d.validateWorkflows()
	if err != nil {
		return validationErrors, err
	}
	for _, v := range verrs {
		validationErrors = append(validationErrors, v)
	}

	return validationErrors, nil
}

func (d *DotGithub) GetAction(n string) *action.Action {
	return d.Actions[n]
}

func (d *DotGithub) GetExternalAction(n string) *action.Action {
	return d.ExternalActions[n]
}

func (d *DotGithub) IsWorkflowJobStepOutputExist(action string, job string, step string, output string) bool {
	if d.Workflows[action] != nil && d.Workflows[action].Jobs[job] != nil {
		if d.Workflows[action].Jobs[job].IsStepOutputExist(step, output, d) == 0 {
			return true
		}
	}
	return false
}

func (d *DotGithub) IsEnvExistInWorkflowOrItsJob(action string, job string, env string) bool {
	if d.Workflows[action] != nil && d.Workflows[action].Jobs[job] != nil {
		if d.Workflows[action].Env[env] != "" || d.Workflows[action].Jobs[job].Env[env] != "" {
			return true
		}
	}
	return false
}

func (d *DotGithub) IsVarsFileExist() bool {
	if d.VarsFile != "" {
		return true
	}
	return false
}

func (d *DotGithub) IsSecretsFileExist() bool {
	if d.SecretsFile != "" {
		return true
	}
	return false
}

func (d *DotGithub) IsVarExist(n string) bool {
	return d.Vars[n]
}

func (d *DotGithub) IsSecretExist(n string) bool {
	return d.Secrets[n]
}
