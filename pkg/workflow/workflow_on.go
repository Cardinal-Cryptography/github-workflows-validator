package workflow

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
		validationErrors = wo.appendErrs(validationErrors, verrs)
	}
	if wo.WorkflowDispatch != nil {
		verrs, err := wo.WorkflowDispatch.Validate(workflow)
		if err != nil {
			return validationErrors, err
		}
		validationErrors = wo.appendErrs(validationErrors, verrs)
	}
	return validationErrors, nil
}

func (wo *WorkflowOn) appendErr(list []string, err string) []string {
	if err != "" {
		list = append(list, err)
	}
	return list
}

func (wo *WorkflowOn) appendErrs(list []string, errs []string) []string {
	if len(errs) > 0 {
		for _, err := range errs {
			list = wo.appendErr(list, err)
		}
	}
	return list
}
