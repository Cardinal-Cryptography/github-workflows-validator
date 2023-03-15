package main

type WorkflowCall struct {
	Inputs map[string]*WorkflowInput `yaml:"inputs"`
}

func (wc *WorkflowCall) Validate(workflow string) ([]string, error) {
	var validationErrors []string
	if wc.Inputs != nil {
		for inputName, input := range wc.Inputs {
			verrs, err := input.Validate(workflow, "call", inputName)
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
