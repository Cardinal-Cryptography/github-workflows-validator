#!/usr/bin/python

import sys
import os
import yaml
import re


def exit_with_error(s):
  sys.stderr.write(s+'\n')
  sys.exit(1)


def print_warning(s):
  sys.stderr.write('!!! '+s+'\n')


def print_info(s):
  sys.stdout.write('*** '+s+'\n')


def exit_if_invalid_env_vars(vars):
  if len(vars) == 0:
    vars = ['DOT_GITHUB_PATH']
  for v in vars:
    if v not in os.environ:
      exit_with_error(v+' env var is missing')


def exit_if_invalid_path():
  if not os.environ['DOT_GITHUB_PATH'].startswith('/'):
    exit_with_error('DOT_GITHUB_PATH must be an absolute path')

  if not os.path.isdir(os.environ['DOT_GITHUB_PATH']):
    exit_with_error('Directory from DOT_GITHUB_PATH does not exist')




def is_lowercase_alphanumeric_with_hyphens(s):
  return re.match(r'^[a-z0-9][a-z0-9\-]+$', s)

def is_uppercase_alphanumeric_with_underscores(s):
  return re.match(r'^[A-Z][A-Z0-9_]+$', s)


def append_error_if_dict_key_missing(dict, keys, errors, err_suffix = ''):
  for k in keys:
    if k not in dict.keys():
      errors.append("missing field '{0}'{1}".format(k, err_suffix))
  return errors


def append_error_if_dict_key_values_not_lowercase_alphanumeric_with_hyphens(dict, keys, errors):
  for k in dict.keys():
    if (len(keys) == 0 or k in keys) and not is_lowercase_alphanumeric_with_hyphens(dict[k]):
      errors.append("invalid field '{0}' - should be lowercase alphanumeric with hyphens".format(k))
  return errors


def append_error_if_dict_key_values_contains_string(dict, keys, string, errors):
  for k in keys:
    if k in dict.keys() and string in dict[k]:
      errors.append("invalid field '{0}' - should not contain '{1}'".format(k, string))
  return errors


def append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(dict, errors, err_suffix = ''):
  for k in dict.keys():
    if not is_lowercase_alphanumeric_with_hyphens(k):
      errors.append("invalid field '{0}'{1} - should be lowercase alphanumeric with hyphens".format(k, err_suffix))
  return errors

# This function matches any ${{ VAR_TYPE.* }} and checks if its name is correct.  It can be used for both
# vars, secrets and env.
def append_error_if_repo_var_not_uppercase_alphanumeric_with_underscore(s, types, errors):
  for type in types:
    names = re.findall(r"\${{[ ]*%s\.([a-zA-Z0-9\-_]+)[ ]*}}" % type, s, re.M)
    for name in names:
      if not is_uppercase_alphanumeric_with_underscores(name):
        errors.append("invalid varname of '{0}.{1}' - should be uppercase alphanumeric with underscores".format(type, name))
  return errors


def get_action_dirnames():
  actions_path = os.path.join(os.environ['DOT_GITHUB_PATH'], 'actions')
  if not os.path.isdir(actions_path):
    return []

  return [ f.name for f in os.scandir(actions_path) if f.is_dir() ]


def get_workflow_filenames():
  workflows_path = os.path.join(os.environ['DOT_GITHUB_PATH'], 'workflows')
  if not os.path.isdir(workflows_path):
    return []

  return [ f.name for f in os.scandir(workflows_path) if f.is_file() and (f.name.endswith('.yaml') or f.name.endswith('.yml')) ]


def get_workflow_yaml(w):
  workflow_path = os.path.join(os.environ['DOT_GITHUB_PATH'], 'workflows', w)
  f = open(workflow_path)
  c = f.read()
  f.close()
  return c

def get_action_yaml(a):
  action_yml = os.path.join(os.environ['DOT_GITHUB_PATH'], 'actions', a, 'action.yml')
  invalid_action_yaml = os.path.join(os.environ['DOT_GITHUB_PATH'], 'actions', a, 'action.yaml')
  action_yml_exists = os.path.isfile(action_yml)
  invalid_action_yaml_exists = os.path.isfile(invalid_action_yaml)
  if not action_yml_exists and not invalid_action_yaml_exists:
    print_warning("cannot validate action {0} as both action.yml and action.yaml are not found".format(a))
    return
  if not action_yml_exists and invalid_action_yaml_exists:
    print_warning("validating action {0} from action.yaml though it should be action.yml...".format(a))
    action_yml = invalid_action_yaml

  f = open(action_yml)
  c = f.read()
  f.close()
  return c




def _get_job_errors(job_dict):
  errors = []
  # VALIDATION: step must have a 'name'
  errors = append_error_if_dict_key_missing(job_dict, ['name'], errors)
  # VALIDATION: if step 'id' exists, it should be lowercase alphanumeric with hyphens
  errors = append_error_if_dict_key_values_not_lowercase_alphanumeric_with_hyphens(job_dict, ['id'], errors)

  if 'uses' not in job_dict.keys():
    # VALIDATION: if no 'uses' then 'runs-on' should exist
    errors = append_error_if_dict_key_missing(job_dict, ['runs-on'], errors, " when no 'uses'")
    # VALIDATION: 'runs-on' should not contain latest
    errors = append_error_if_dict_key_values_contains_string(job_dict, ['runs-on'], "latest", errors)

  if 'uses' not in job_dict.keys():
    steps_errors = _get_job_steps_errors(job_dict['steps'])
    if len(steps_errors) > 0:
      for e in steps_errors:
        errors.append('steps -> {0}'.format(e))

  return errors


def _get_job_step_outputs(steps_dict):
  steps_outputs = {} 
  for s in steps_dict:
    if 'id' in s.keys():
      steps_outputs[s['id']] = {}
      # If step has id and it has 'run' key, we parse out the 'echo "name=.*" >> $GITHUB_OUTPUT' strings
      if 'run' in s.keys():
        steps_outputs[s['id']]['__run_found'] = True
        github_outputs = re.findall(r'echo[ ]+["]{0,1}([a-zA-Z0-9\-_]+)=.*["]{0,1}[ ]+>>[ ]+\$GITHUB_OUTPUT', s['run'], re.M)
        for o in github_outputs:
          steps_outputs[s['id']][o] = True
  return steps_outputs


def _get_job_steps_errors(steps_dict):
  errors = []
  job_step_outputs = _get_job_step_outputs(steps_dict)

  i = 0
  for step_dict in steps_dict:
    step_errors = _get_step_errors(step_dict, job_step_outputs)
    if len(step_errors) > 0:
      for e in step_errors:
        errors.append('step {0} -> {1}'.format(i, e))
    i+=1
  return errors


def _get_step_errors(step_dict, job_step_outputs):
  errors = []
  # VALIDATION: step must have a 'name'
  errors = append_error_if_dict_key_missing(step_dict, ['name'], errors)
  # VALIDATION: if step 'id' exists, it should be lowercase alphanumeric with hyphens
  errors = append_error_if_dict_key_values_not_lowercase_alphanumeric_with_hyphens(step_dict, ['id'], errors)

  # VALIDATION: 'name' must be the first field
  # Requires >Python 3.7: https://mail.python.org/pipermail/python-dev/2017-December/151283.html
  if list(step_dict.keys())[0] != 'name':
    errors.append("first field must be 'name'")

  # VALIDATION: Calls in 'run' to non-existinging step outputs
  if 'run' in step_dict.keys():
    if isinstance(step_dict['run'], str):
      missing = _get_missing_step_outputs(step_dict['run'], job_step_outputs)
      if len(missing) > 0:
        for m in missing:
          errors.append("call to missing step output {0} in 'run' (or deprecated method for setting output used)".format(m))

  # VALIDATION: Calls in 'env' or 'with' to non-existinging step outputs
  for key_to_check in ['env', 'with']:
    if key_to_check in step_dict.keys():
      for subkey in step_dict[key_to_check].keys():
        if isinstance(step_dict[key_to_check][subkey], str):
          missing = _get_missing_step_outputs(step_dict[key_to_check][subkey], job_step_outputs)
          if len(missing) > 0:
            for m in missing:
              errors.append("call to missing step output {0} in '{1}.{2}' (or deprecated method for setting output used)".format(m, key_to_check, subkey))
  return errors


def _get_missing_step_outputs(contents, job_step_outputs):
  missing = []
  var_steps_outputs = re.findall(r'\${{[ ]*steps\.([a-zA-Z0-9\-_]+)\.outputs\.([a-zA-Z0-9\-_]+)[ ]*}}', contents, re.M)
  for f in var_steps_outputs:
    if f[0] not in job_step_outputs:
      missing.append(f[0])
      continue
    if '__run_found' in job_step_outputs[f[0]] and f[1] not in job_step_outputs[f[0]]:
      missing.append(f[0]+'.'+f[1])
  return missing


def get_errors_from_workflow(w):
  errors = []
  s = get_workflow_yaml(w)
  y = yaml.safe_load(s)

  # VALIDATION: workflow must have a 'name'
  errors = append_error_if_dict_key_missing(y, ['name'], errors)
  # VALIDATION: job name should be lowercase alphanumeric with hyphens
  errors = append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(y['jobs'], errors)
  
  job_names = y['jobs'].keys()
  # VALIDATION: if there is only one job in the workflow then it should be named 'main'
  if len(job_names) == 1 and list(job_names)[0] != 'main':
    errors.append("job '{0}' is the only job in the workflow and should be named 'main'".format(list(job_names)[0]))

  # Loop through jobs and validate them
  for job_name in job_names:
    job_errors = _get_job_errors(y['jobs'][job_name])
    if len(job_errors) > 0:
      for e in job_errors:
        errors.append("job {0} -> {1}".format(job_name, e))

  # VALIDATION: vars, secrets and env vars must be uppercase ALPHANUMERIC with underscode
  errors = append_error_if_repo_var_not_uppercase_alphanumeric_with_underscore(s, ['env', 'var', 'secrets'], errors)

  return errors


def get_errors_from_action(a):
  errors = []
  s = get_action_yaml(a)
  y = yaml.safe_load(s)

  # VALIDATION: action must have 'name' and 'description' fields
  errors = append_error_if_dict_key_missing(y, ['name', 'description'], errors)
  # VALIDATION: inputs and output must be lowercase alphanumeric with hyphens
  if 'inputs' in y.keys():
    errors = append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(y['inputs'], errors, " in 'inputs'")
  if 'outputs' in y.keys():
    errors = append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(y['outputs'], errors, " in 'outputs'")

  # VALIDATION: inputs and outputs must have 'description' field
  if 'inputs' in y.keys():
    for i in y['inputs'].keys():
      errors = append_error_if_dict_key_missing(y['inputs'][i], ['description'], errors, " in 'inputs.{0}'".format(i))
  if 'outputs' in y.keys():
    for o in y['outputs'].keys():
      errors = append_error_if_dict_key_missing(y['outputs'][o], ['description'], errors, " in 'outputs.{0}'".format(o))

  # VALIDATION: vars, secrets and env vars must be uppercase ALPHANUMERIC with underscores
  errors = append_error_if_repo_var_not_uppercase_alphanumeric_with_underscore(s, ['env', 'var', 'secrets'], errors)

  return errors


def get_errors_from_action_filenames(dirnames):
  errors = []
  for dir in dirnames:
    # VALIDATION: action should have action.yml (and not .yaml)
    action_yml = os.path.join(os.environ['DOT_GITHUB_PATH'], 'actions', dir, 'action.yml')
    invalid_action_yaml = os.path.join(os.environ['DOT_GITHUB_PATH'], 'actions', dir, 'action.yaml')
    action_yml_exists = os.path.isfile(action_yml)
    invalid_action_yaml_exists = os.path.isfile(invalid_action_yaml)
    if not action_yml_exists and invalid_action_yaml_exists:
      errors.append("invalid extension - expected {0} but found {1}".format(action_yml, invalid_action_yaml))
    if not action_yml_exists and not invalid_action_yaml_exists:
      errors.append("missing or invalid action file - {0} (nor invalid action.yaml) not found".format(action_yml))
    if action_yml_exists and invalid_action_yaml_exists:
      errors.append("duplicated action file - found both valid {0} and invalid {1}".format(action_yml, invalid_action_yaml))

    # VALIDATION: action dir name should have lower case alphanumeric and hyphen only
    if not is_lowercase_alphanumeric_with_hyphens(dir):
      errors.append("invalid action dir name '{0}' - should be lowercase alphanumeric with hyphens".format(dir))

  return errors


def get_errors_from_workflow_filenames(filenames):
  errors = []
  for f in filenames:
    # VALIDATION: .yml extension
    if not f.endswith('.yml'):
      errors.append("workflow {0} should end with .yml".format(f))

    # VALIDATION: should not contain underscore
    if '_' in f:
      errors.append("workflow {0} filename should not contain underscore - use hyphens".format(f))

    # VALIDATION: workflow filename can start with underscore and then it should be lower case alphanumeric and hyphen only
    if not re.match(r'^[_]{0,1}[a-z0-9][a-z0-9\-]+\.yml$', f):
      errors.append("invalid workflow filename '{0}' - should be lower alphanumeric with hyphen, optionally starting with underscore when it is sub-workflow, and ending with .yml".format(f))

  return errors


def main():
  exit_if_invalid_env_vars([])
  exit_if_invalid_path()

  action_dirnames = get_action_dirnames()
  workflow_filenames = get_workflow_filenames()
  print_info('Found action dirnames: ' + ', '.join(action_dirnames))
  print_info('Found workflow filenames: ' + ', '.join(workflow_filenames))

  # Validate action dir name and file inside
  errors = get_errors_from_action_filenames(action_dirnames)
  if len(errors) > 0:
    for err in errors:
      print_warning('action filenames -> {0}'.format(err))

  # Validate workflow file names
  errors = get_errors_from_workflow_filenames(workflow_filenames)
  if len(errors) > 0:
    for err in errors:
      print_warning('workflow filenames -> {0}'.format(err))

  # Loop through actions and validate them
  for a in action_dirnames:
    action_errors = get_errors_from_action(a)
    if len(action_errors) > 0:
      for err in action_errors:
        print_warning('action {0} -> {1}'.format(a, err))
  
  # Loop through workflows and validate them
  for w in workflow_filenames:
    workflow_errors = get_errors_from_workflow(w)
    if len(workflow_errors) > 0:
      for err in workflow_errors:
        print_warning('workflow {0} -> {1}'.format(w, err))

  # TODO: make script fail when there are any errors


if __name__ == "__main__":
  main()
