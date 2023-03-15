package main

type WorkflowDispatch struct {
	Inputs map[string]*WorkflowInput `yaml:"inputs"`
}

func (wd *WorkflowDispatch) Validate(workflow string) ([]string, error) {
	var validationErrors []string
	if wd.Inputs != nil {
		for inputName, input := range wd.Inputs {
			verrs, err := input.Validate(workflow, "dispatch", inputName)
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
