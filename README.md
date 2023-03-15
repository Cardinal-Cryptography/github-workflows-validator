# docker-github-workflows-validator
Quick tool to validate workflows and action in .github directory

## Building
Run `go build -o github-workflows-validator` to compile the binary.

### Building docker image
To build the docker image, use the following command.

    docker build -t github-workflows-validator .


## Running
Check below help message for `validate` command:

    Usage:  docker-github-workflows-validator validate [FLAGS]

    Runs the validation on files from a specified directory

    Required flags:
      -p,    --path         Path to .github directory

Use `-p` argument to point to `.github` directories.  The tool will search for any actions in the `actions`
directory, where each action is in its own sub-directory and its filename is either `action.yaml` or
`action.yml`.  And, it will search for workflows' `*.yml` and `*.yaml` files in `workflows` directory.


### Using docker image
Note that the image has to be present, either built or pulled from the registry.
Replace path to the .github directory.

    docker run --rm --name tmp-gh-wf-validator \
      -v /Users/me/my-repo/.github:/dot-github \
      github-workflows-validator \
	  validate -p /dot-github


## Exit code
Currently, tool always exit with code 0.  To check if there are any errors, please use `grep` to filter
the output for errors.

## TODO

* workflow jobs - validate needs
