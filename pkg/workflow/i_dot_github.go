package workflow

import (
	"github.com/Cardinal-Cryptography/github-actions-validator/pkg/action"
)

type IDotGithub interface {
	action.IDotGithub
	IsVarsFileExist() bool
	IsSecretsFileExist() bool
	IsVarExist(n string) bool
	IsSecretExist(n string) bool
}
