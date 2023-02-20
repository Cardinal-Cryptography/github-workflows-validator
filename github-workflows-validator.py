#!/usr/bin/python

import sys
import os
import yaml
import re


def err(s):
  sys.stderr.write(s+'\n')
  sys.exit(1)


def v_inf(s):
  sys.stderr.write('!!! '+s+'\n')


def dbg(s):
  sys.stdout.write('*** '+s+'\n')


def check_env_vars(vars):
  if len(vars) == 0:
    vars = ['DOT_GITHUB_PATH']
  for v in vars:
    if v not in os.environ:
      err(v+' env var is missing')


def check_if_path_exists():
  if not os.environ['DOT_GITHUB_PATH'].startswith('/'):
    err('DOT_GITHUB_PATH must be an absolute path')

  if not os.path.isdir(os.environ['DOT_GITHUB_PATH']):
    err('Directory from DOT_GITHUB_PATH does not exist')


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


def get_workflow_yaml_dict(w):
  workflow_path = os.path.join(os.environ['DOT_GITHUB_PATH'], 'workflows', w)
  f = open(workflow_path)
  c = f.read()
  f.close()
  return yaml.safe_load(c)


def _get_job_errors(job_dict):
  errors = []
  # VALIDATION: Check if job has a name
  if 'name' not in job_dict.keys():
    errors.append('missing name')

  # VALIDATION: Check if job runs-on does not contain latest
  if 'uses' not in job_dict.keys() and ('runs-on' not in job_dict.keys() or 'latest' in job_dict['runs-on']):
    errors.append("runs-on is missing or contains 'latest'")

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
  if 'name' not in step_dict.keys():
    errors.append('missing name')
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
  y = get_workflow_yaml_dict(w)
  job_names = y['jobs'].keys()
  # TODO: validate job names
  for job_name in job_names:
    job_errors = _get_job_errors(y['jobs'][job_name])
    if len(job_errors) > 0:
      for e in job_errors:
        errors.append("job {0} -> {1}".format(job_name, e))
  return errors


def main():
  check_env_vars([])
  check_if_path_exists()

  action_dirnames = get_action_dirnames()
  workflow_filenames = get_workflow_filenames()
  dbg('Found action dirnames: ' + ', '.join(action_dirnames))
  dbg('Found workflow filenames: ' + ', '.join(workflow_filenames))

  # TODO: validate action dirnames
  # TODO: validate action.yaml files
  # TODO: validate workflow filenames

  for w in workflow_filenames:
    workflow_errors = get_errors_from_workflow(w)
    if len(workflow_errors) > 0:
      for err in workflow_errors:
        v_inf('workflow {0} -> {1}'.format(w, err))

  # TODO: make script fail when there are any errors


if __name__ == "__main__":
  main()
