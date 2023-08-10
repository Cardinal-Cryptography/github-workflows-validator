package main

type IDotGithub interface {
	GetAction(n string) *Action
	DownloadExternalAction(path string) error
	GetExternalAction(n string) *Action
	GetWorkflowJob(action string, job string) *WorkflowJob
	GetWorkflowJobEnv(action string, job string, env string) string
	GetWorkflowEnv(action string, env string) string
	IsVarsFileExist() bool
	IsSecretsFileExist() bool
	IsVarExist(n string) bool
	IsSecretExist(n string) bool
}
