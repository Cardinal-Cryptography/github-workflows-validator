package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DotGithub struct {
	Path            string
	Actions         map[string]*Action
	ExternalActions map[string]*Action
	Workflows       map[string]*Workflow
}

func (d *DotGithub) InitFiles() error {
	if d.Path == "" {
		return nil
	}

	d.getActions()
	d.getWorkflows()

	for _, a := range d.Actions {
		err := a.Init(false)
		if err != nil {
			return err
		}
	}
	for _, w := range d.Workflows {
		err := w.Init()
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DotGithub) DownloadExternalAction(path string) error {
	if d.ExternalActions == nil {
		d.ExternalActions = map[string]*Action{}
	}
	if d.ExternalActions[path] != nil {
		return nil
	}
	repoVersion := strings.Split(path, "@")
	actionURLPrefix := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", repoVersion[0], repoVersion[1])

	req, err := http.NewRequest("GET", actionURLPrefix+"/action.yml", strings.NewReader(""))
	if err != nil {
		return err
	}
	c := &http.Client{}
	resp, err := c.Do(req)

	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		req, err = http.NewRequest("GET", actionURLPrefix+"/action.yaml", strings.NewReader(""))
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
	b, _ := ioutil.ReadAll(resp.Body)

	d.ExternalActions[path] = &Action{
		Path:    path,
		DirName: "",
		Raw:     b,
	}
	err = d.ExternalActions[path].Init(true)
	if err != nil {
		return err
	}
	return nil
}

func (d *DotGithub) getActions() {
	d.Actions = map[string]*Action{}
	actionsPath := filepath.Join(d.Path, "actions")
	entries, err := os.ReadDir(actionsPath)
	if err != nil {
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
		d.Actions[e.Name()] = &Action{
			Path:    actionYMLPath,
			DirName: e.Name(),
		}
	}
}

func (d *DotGithub) getWorkflows() {
	d.Workflows = map[string]*Workflow{}
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
		d.Workflows[e.Name()] = &Workflow{
			Path: entryPath,
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
