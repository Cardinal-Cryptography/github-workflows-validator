package main

type WorkflowOn struct {
	WorkflowCall     *WorkflowCall     `yaml:"workflow_call"`
	WorkflowDispatch *WorkflowDispatch `yaml:"workflow_dispatch"`
}

func (wo *WorkflowOn) Validate(workflow string) ([]string, error) {
	var validationErrors []string
	if wo.WorkflowCall != nil {
		verrs, err := wo.WorkflowCall.Validate(workflow)
		if err != nil {
			return validationErrors, err
		}
		if len(verrs) > 0 {
			for _, verr := range verrs {
				validationErrors = append(validationErrors, verr)
			}
		}
	}
	if wo.WorkflowDispatch != nil {
		verrs, err := wo.WorkflowDispatch.Validate(workflow)
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
