# github-actions-validator
Quick tool to validate workflows and actions in .github directory

## Checks
See the checks that are performed on all the workflow and action files.  These are separate into errors
and warnings.  Each check has a code where as one starting with `E` indicates an error, `N` indicates
a warning about invalid naming convention.  Additionally, code will contain either `A` if it is an 
action where the issue is found, and `W` if issue occurs in a workflow.
### Errors
| Code | Description |
|------|-------------|
| EA809 | Called step with id '%s' does not exist |
| EA811 | Called step with id '%s' output '%s' does not exist |
| EW203 | Job '%s' has invalid value '%s' in 'needs' field |
| EW201 | Called variable '%s' is invalid |
| EW202 | Called input '%s' does not exist |
| EW203 | Job '%s' has invalid value '%s' in 'needs' field |
| EW801 | Path to external action '%s' is invalid |
| EW802 | Path to local action '%s' is invalid |
| EW803 | Call to non-existing local action '%s' |
| EW804 | Required input '%s' missing for local action '%s' |
| EW805 | Input '%s' does not exist in local action '%s' |
| EW806 | Required input '%s' missing for external action '%s' |
| EW807 | Input '%s' does not exist in external action '%s' |
| EW808 | Call to non-existing external action '%s' |
| EW809 | Called step with id '%s' does not exist |
| EW810 | Called step with id '%s' does not exist |
| EW811 | Called step with id '%s' output '%s' does not exist |

### Naming convention warnings

| Code | Description |
|------|-------------|
| NA101 | Action directory name should contain lowercase alphanumeric characters and hyphens only |
| NA102 | Action file name should have .yml extension |
| NA103 | Action name is empty |
| NA104 | Action description is empty |
| NA301 | Action input name should contain lowercase alphanumeric characters and hyphens only |
| NA302 | Action input must have a description |
| NA501 | Action output name should contain lowercase alphanumeric characters and hyphens only |
| NA502 | Action output must have a description |
| NW101 | Workflow file name should contain alphanumeric characters and hyphens only |
| NW102 | Workflow file name should have .yml extension |
| NW103 | Env variable name '%s' should contain uppercase alphanumeric characters and underscore only |
| NW104 | Workflow name is empty |
| NW105 | Workflow description is empty |
| NW106 | When workflow has only one job, it should be named 'main' |
| NW107 | Called variable name '%s' should contain uppercase alphanumeric characters and underscore only |
| NW301 | Workflow input name should contain lowercase alphanumeric characters and hyphens only |
| NW302 | Workflow input must have a description |
| NW501 | Workflow job name should contain lowercase alphanumeric characters and hyphens only |
| NW502 | Env variable name '%s' should contain uppercase alphanumeric characters and underscore only |
| NW701 | Env variable name '%s' should contain uppercase alphanumeric characters and underscore only |


## Building
Run `go build -o github-actions-validator` to compile the binary.

### Building docker image
To build the docker image, use the following command.

    docker build -t github-actions-validator .


## Running
Check below help message for `validate` command:

    Usage:  github-actions-validator validate [FLAGS]

    Runs the validation on files from a specified directory

    Required flags:
      -p,    --path         Path to .github directory

Use `-p` argument to point to `.github` directories.  The tool will search for any actions in the `actions`
directory, where each action is in its own sub-directory and its filename is either `action.yaml` or
`action.yml`.  And, it will search for workflows' `*.yml` and `*.yaml` files in `workflows` directory.


### Using docker image
Note that the image has to be present, either built or pulled from the registry.
Replace path to the .github directory.

    docker run --rm --name tmp-gha-validator \
      -v /Users/me/my-repo/.github:/dot-github \
      github-actions-validator \
	  validate -p /dot-github


## Exit code
Currently, tool always exit with code 0.  To check if there are any errors, please use `grep` to filter
the output for errors.

## TODO

* warning about reference to not defined env (might be defined outside of code, hence just warning)
