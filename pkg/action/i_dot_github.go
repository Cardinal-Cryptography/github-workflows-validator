package action

type IDotGithub interface {
	GetAction(n string) *Action
	DownloadExternalAction(path string) error
	GetExternalAction(n string) *Action

	IsWorkflowJobStepOutputExist(action string, job string, step string, output string) bool
	IsEnvExistInWorkflowOrItsJob(action string, job string, env string) bool
}
