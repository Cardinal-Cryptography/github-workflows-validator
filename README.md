# docker-github-workflows-validator
Quick script to validate github workflows

## Building docker image
Run the following command to build the image.

    docker build -t github-workflows-validator -f Dockerfile .


## Running

### Command-line
Python 3 is required to run the script.  Replace path to the .github directory below.

    DOT_GITHUB_PATH=/Users/me/my-repo/.github python3 github-workflows-validator.py

### Using docker image
Note that the image has to be present, either built or pulled from the registry.
Replace path to the .github directory.

    docker run --rm --name tmp-gha-validator \
      -v /Users/me/my-repo/.github:/dot-github \
      -e DOT_GITHUB_PATH=/dot-github \
      github-workflows-validator

