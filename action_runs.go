package main

import (
	"regexp"
	"strconv"
	"strings"
)

type ActionRuns struct {
	Using string        `yaml:"using"`
	Steps []*ActionStep `yaml:"steps"`
}

func (ar *ActionRuns) IsStepExist(id string) bool {
	for _, s := range ar.Steps {
		if s.Id == id {
			return true
		}
	}
	return false
}

func (ar *ActionRuns) IsStepOutputExist(step string, output string, d *DotGithub) int {
	for _, s := range ar.Steps {
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

func (ar *ActionRuns) Validate(dirName string, d *DotGithub) ([]string, error) {
	var validationErrors []string
	if ar.Steps != nil {
		for i, s := range ar.Steps {
			verrs, err := s.Validate(dirName, strconv.Itoa(i), d)
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
