#!/usr/bin/python

import sys
import os
import re
import yaml
import requests  # pylint: disable=import-error


cache_external_actions = {}


def exit_with_error(s):
    sys.stderr.write(s+'\n')
    sys.exit(1)


def print_warning(s):
    sys.stderr.write('!!! '+s+'\n')


def print_info(s):
    sys.stdout.write('*** '+s+'\n')


def exit_if_invalid_env_vars(env_vars):
    if len(env_vars) == 0:
        env_vars = ['DOT_GITHUB_PATH']
    for v in env_vars:
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


def append_error_if_dict_key_missing(dict_to_search, required_keys, error_list, err_suffix=''):
    for k in required_keys:
        if k not in dict_to_search.keys():
            error_list.append(f"missing field '{k}'{err_suffix}")
    return error_list


def append_error_if_dict_key_values_not_lowercase_alphanumeric_with_hyphens(dict_to_search, keys_to_validate, error_list):
    for k in dict_to_search.keys():
        if (len(keys_to_validate) == 0 or k in keys_to_validate) and not is_lowercase_alphanumeric_with_hyphens(dict_to_search[k]):
            error_list.append(f"invalid field '{k}' - should be lowercase alphanumeric with hyphens")
    return error_list


def append_error_if_dict_key_values_contains_string(dict_to_search, keys_to_validate, string, error_list):
    for k in keys_to_validate:
        if k in dict_to_search.keys() and string in dict_to_search[k]:
            error_list.append(f"invalid field '{k}' - should not contain '{string}'")
    return error_list


def append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(dict_to_search, error_list, err_suffix=''):
    for k in dict_to_search.keys():
        if not is_lowercase_alphanumeric_with_hyphens(k):
            error_list.append(
                f"invalid field '{k}'{err_suffix} - should be lowercase alphanumeric with hyphens")
    return error_list

# This function matches any ${{ VAR_TYPE.* }} and checks if its name is correct.  It can be used for both
# vars, secrets and env.


def append_error_if_repo_var_not_uppercase_alphanumeric_with_underscore(s, types, errors):
    for var_type in types:
        names = re.findall(
            r"\${{[ ]*%s\.([a-zA-Z0-9\-_]+)[ ]*}}" % var_type, s, re.M)  # pylint: disable=consider-using-f-string
        for name in names:
            if not is_uppercase_alphanumeric_with_underscores(name):
                errors.append(
                    f"invalid varname of '{var_type}.{name}' - should be uppercase alphanumeric with underscores")
    return errors


def append_error_if_var_not_in_list(s, var_type, existing_vars, errors):
    names = re.findall(r"\${{[ ]*%s\.([a-zA-Z0-9\-_]+)[ ]*}}" %  # pylint: disable=consider-using-f-string
                       var_type, s, re.M)
    for name in names:
        if name not in existing_vars:
            errors.append(f"call to non-existing varname of '{var_type}.{name}'")
    return errors


def append_error_if_step_output_not_in_list(s, step_list, step_output_list, errors, err_suffix=''):
    var_steps_outputs = re.findall(
        r'\${{[ ]*steps\.([a-zA-Z0-9\-_]+)\.outputs\.([a-zA-Z0-9\-_]+)[ ]*}}', s, re.M)
    for f in var_steps_outputs:
        if f[0] not in step_list:
            errors.append(f"call to missing step '{f[0]}'{err_suffix}")
        if f"{f[0]}.{f[1]}" not in step_output_list and f"{f[0]}.*" not in step_output_list:
            errors.append(
                f"call to missing step output (or deprecated method for setting output used) '{f[0]}.{f[1]}'{err_suffix}")
    return errors


def append_error_if_dict_steps_refer_nonexisting_local_action(dict_to_search, action_inputs_and_outputs, errors):
    if 'steps' in dict_to_search:  # pylint: disable=too-many-nested-blocks
        i = 0
        for s in dict_to_search['steps']:
            if 'uses' in s.keys() and s['uses'].startswith('./.github/'):
                # VALIDATION: check if local action exists
                if not re.match(r'\.\/\.github\/actions\/[a-z0-9\-]+', s['uses']):
                    errors.append(
                        f"step {i} -> invalid value for 'uses' referring local action '{s['uses']}'")
                if s['uses'] not in [f"./.github/actions/{d}" for d in list(action_inputs_and_outputs.keys())]:
                    errors.append(
                        f"step {i} -> 'uses' references non-existing local action '{s['uses']}'")
                    continue
                action_name = s['uses'].replace('./.github/actions/', '')
                # VALIDATION: check if all required fields are passed
                for required_field in action_inputs_and_outputs[action_name]['inputs']['required']:
                    if 'with' not in s or required_field not in s['with']:
                        errors.append(
                            f"step {i} -> required field '{required_field}' missing for action '{action_name}")
                # VALIDATION: check if all passed fields are valid
                if 'with' in s:
                    for uses_field in s['with']:
                        if uses_field not in action_inputs_and_outputs[action_name]['inputs']['required'] and uses_field not in action_inputs_and_outputs[action_name]['inputs']['optional']:
                            errors.append(
                                f"step {i} -> field '{uses_field}' cannot be found in inputs of action '{action_name}'")
            i += 1
    return errors


def append_error_if_dict_steps_refer_nonexisting_external_action(dict_to_search, errors):
    if 'steps' in dict_to_search:  # pylint: disable=too-many-nested-blocks
        i = 0
        for s in dict_to_search['steps']:
            if 'uses' in s.keys() and re.match(r'[a-zA-Z0-9\-_]+\/[a-zA-Z0-9\-_]+@[a-zA-Z0-9\-_]+', s['uses']):
                # VALIDATION: check if external action exists by taking the name and trying to download the action yaml file
                inputs_and_outputs = get_external_action_inputs_and_outputs(
                    s['uses'])
                if s['uses'] not in inputs_and_outputs:
                    errors.append(
                        f"step {i} -> 'uses' references non-existing external action '{s['uses']}'")

                action_name = s['uses']
                # VALIDATION: check if all required fields are passed
                for required_field in inputs_and_outputs[action_name]['inputs']['required']:
                    if 'with' not in s or required_field not in s['with']:
                        errors.append(
                            f"step {i} -> required field '{required_field}' missing for action '{action_name}")
                # VALIDATION: check if all passed fields are valid
                if 'with' in s:
                    for uses_field in s['with']:
                        if uses_field not in inputs_and_outputs[action_name]['inputs']['required'] and uses_field not in inputs_and_outputs[action_name]['inputs']['optional']:
                            errors.append(
                                f"step {i} -> field '{uses_field}' cannot be found in inputs of action '{action_name}'".format(i, uses_field, action_name))
            i += 1

    return errors


def get_action_dirnames():
    actions_path = os.path.join(os.environ['DOT_GITHUB_PATH'], 'actions')
    if not os.path.isdir(actions_path):
        return []
    return [f.name for f in os.scandir(actions_path) if f.is_dir()]


def get_workflow_filenames():
    workflows_path = os.path.join(os.environ['DOT_GITHUB_PATH'], 'workflows')
    if not os.path.isdir(workflows_path):
        return []
    return [f.name for f in os.scandir(workflows_path) if f.is_file() and (f.name.endswith('.yaml') or f.name.endswith('.yml'))]


def get_workflow_yaml(w):
    workflow_path = os.path.join(os.environ['DOT_GITHUB_PATH'], 'workflows', w)
    with open(workflow_path, encoding='utf-8') as f:
        c = f.read()
    return c


def get_action_yaml(a):  # pylint: disable=inconsistent-return-statements
    action_yml = os.path.join(
        os.environ['DOT_GITHUB_PATH'], 'actions', a, 'action.yml')
    invalid_action_yaml = os.path.join(
        os.environ['DOT_GITHUB_PATH'], 'actions', a, 'action.yaml')
    action_yml_exists = os.path.isfile(action_yml)
    invalid_action_yaml_exists = os.path.isfile(invalid_action_yaml)

    if not action_yml_exists and not invalid_action_yaml_exists:
        print_warning(
            f"cannot validate action {a} as both action.yml and action.yaml are not found")
        return

    if not action_yml_exists and invalid_action_yaml_exists:
        print_warning(
            f"validating action {a} from action.yaml though it should be action.yml...")
        action_yml = invalid_action_yaml

    with open(action_yml, encoding='utf-8') as f:
        c = f.read()
        return c

    return


def get_external_action_yaml(path):
    [repo, version] = path.split('@')

    if repo in cache_external_actions.keys():  # pylint: disable=consider-iterating-dictionary
        print_info(f"Found external action '{repo}' in already downloaded actions")
        return cache_external_actions[repo]

    action_prefix = f"https://raw.githubusercontent.com/{repo}/{version}"
    c = ''
    resp = requests.get(action_prefix+'/action.yml')
    if resp.status_code == 200:
        c = resp.text
        print_info(f"Downloaded external action from {action_prefix}/action.yml")
    else:
        resp = requests.get(action_prefix+'/action.yaml')
        if resp.status_code == 200:
            c = resp.text
            print_info(f"Downloaded external action from {action_prefix}/action.yaml")
    if c == '':
        print_warning(
            f"neither action.yml nor action.yaml found at {action_prefix}")
    else:
        cache_external_actions[repo] = c
    return c


def get_external_action_inputs_and_outputs(path):
    c = get_external_action_yaml(path)
    if c == '':
        return {}

    actions_with_ios = {}
    y = yaml.safe_load(c)  # pylint: disable=no-member
    actions_with_ios = append_action_inputs_and_outputs(
        actions_with_ios, path, y)
    return actions_with_ios


def _get_job_errors(job_dict, action_inputs_and_outputs):
    errors = []
    # VALIDATION: step must have a 'name'
    errors = append_error_if_dict_key_missing(job_dict, ['name'], errors)
    # VALIDATION: if step 'id' exists, it should be lowercase alphanumeric with hyphens
    errors = append_error_if_dict_key_values_not_lowercase_alphanumeric_with_hyphens(
        job_dict, ['id'], errors)

    if 'uses' not in job_dict.keys():
        # VALIDATION: if no 'uses' then 'runs-on' should exist
        errors = append_error_if_dict_key_missing(
            job_dict, ['runs-on'], errors, " when no 'uses'")
        # VALIDATION: 'runs-on' should not contain latest
        errors = append_error_if_dict_key_values_contains_string(
            job_dict, ['runs-on'], "latest", errors)

    if 'uses' not in job_dict.keys():
        steps_errors = _get_job_steps_errors(job_dict['steps'])
        if len(steps_errors) > 0:
            for e in steps_errors:
                errors.append(f"{e}")

        # VALIDATION: 'uses' in step must refer to existing action and have required fields
        errors = append_error_if_dict_steps_refer_nonexisting_local_action(
            job_dict, action_inputs_and_outputs, errors)
        # VALIDATION: 'uses' in step must refer to existing external action and have required fields
        errors = append_error_if_dict_steps_refer_nonexisting_external_action(
            job_dict, errors)
    return errors


def _get_job_steps_errors(steps_dict):
    errors = []
    # job_step_outputs = _get_job_step_outputs(steps_dict)

    step_list = [s['id'] for s in steps_dict if 'id' in s.keys()]
    step_output_list = []
    for s in steps_dict:
        if 'id' in s.keys():
            if 'run' in s.keys():
                for o in re.findall(r'echo[ ]+["]{0,1}([a-zA-Z0-9\-_]+)=.*["]{0,1}[ ]+>>[ ]+\$GITHUB_OUTPUT', s['run'], re.M):
                    step_output_list.append(f"{s['id']}.{o}")
            else:
                step_output_list.append(f"{s['id']}.*")

    i = 0
    for step_dict in steps_dict:
        step_errors = _get_step_errors(step_dict, step_list, step_output_list)
        if len(step_errors) > 0:
            for e in step_errors:
                errors.append(f"step {i} -> {e}")
        i += 1
    return errors


def _get_step_errors(step_dict, step_list, step_output_list):
    errors = []
    # VALIDATION: step must have a 'name'
    errors = append_error_if_dict_key_missing(step_dict, ['name'], errors)
    # VALIDATION: if step 'id' exists, it should be lowercase alphanumeric with hyphens
    errors = append_error_if_dict_key_values_not_lowercase_alphanumeric_with_hyphens(
        step_dict, ['id'], errors)

    # VALIDATION: 'name' must be the first field
    # Requires >Python 3.7: https://mail.python.org/pipermail/python-dev/2017-December/151283.html
    if list(step_dict.keys())[0] != 'name':
        errors.append("first field must be 'name'")

    # VALIDATION: Calls in 'run' to non-existinging step outputs
    if 'run' in step_dict.keys():
        if isinstance(step_dict['run'], str):
            errors = append_error_if_step_output_not_in_list(
                step_dict['run'], step_list, step_output_list, errors, " in 'run'")

    # VALIDATION: Calls in 'env' or 'with' to non-existinging step outputs
    for key_to_check in ('env', 'with'):
        if key_to_check in step_dict.keys():
            for subkey in step_dict[key_to_check].keys():
                if isinstance(step_dict[key_to_check][subkey], str):
                    errors = append_error_if_step_output_not_in_list(
                        step_dict[key_to_check][subkey], step_list, step_output_list, errors, f" in '{key_to_check}.{subkey}'")
    return errors


def get_errors_from_workflow(w, action_inputs_and_outputs):
    errors = []
    s = get_workflow_yaml(w)
    # 'on:' is replaced with 'True:' because PyYAML is stupid, hence we change it to 'real_on:'
    s = re.sub('^on:.*$', 'real_on:', s, flags=re.MULTILINE)
    y = yaml.safe_load(s)  # pylint: disable=no-member

    # VALIDATION: workflow must have a 'name'
    errors = append_error_if_dict_key_missing(y, ['name'], errors)
    # VALIDATION: job name should be lowercase alphanumeric with hyphens
    errors = append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(
        y['jobs'], errors)

    job_names = y['jobs'].keys()
    # VALIDATION: if there is only one job in the workflow then it should be named 'main'
    if len(job_names) == 1 and list(job_names)[0] != 'main':
        errors.append(f"job '{list(job_names)[0]}' is the only job in the workflow and should be named 'main'")

    # Loop through jobs and validate them
    for job_name in job_names:
        job_errors = _get_job_errors(
            y['jobs'][job_name], action_inputs_and_outputs)
        if len(job_errors) > 0:
            for e in job_errors:
                errors.append(f"job {job_name} -> {e}")

    # VALIDATION: vars, secrets and env vars must be uppercase ALPHANUMERIC with underscode
    errors = append_error_if_repo_var_not_uppercase_alphanumeric_with_underscore(
        s, ['env', 'var', 'secrets'], errors)

    # VALIDATION: call to non-existing inputs
    input_names = list(y['real_on']['workflow_call']['inputs'].keys()) if 'workflow_call' in y['real_on'].keys(
    ) and y['real_on']['workflow_call'] is not None and 'inputs' in y['real_on']['workflow_call'].keys() else []
    if 'workflow_dispatch' in y['real_on'].keys() and y['real_on']['workflow_dispatch'] is not None and 'inputs' in y['real_on']['workflow_dispatch'].keys():
        input_names += list(y['real_on']['workflow_call']['inputs'].keys())
    errors = append_error_if_var_not_in_list(s, 'inputs', input_names, errors)

    return errors


def append_action_inputs_and_outputs(append_to, action_name, action_dict):
    append_to[action_name] = {
        'inputs': {
            'required': [],
            'optional': []
        },
        'outputs': []
    }
    if 'inputs' in action_dict.keys():
        for k, v in action_dict['inputs'].items():
            if 'required' in v and (v['required'] is True or v['required'] == 'true'):
                append_to[action_name]['inputs']['required'].append(k)
            else:
                append_to[action_name]['inputs']['optional'].append(k)
    if 'outputs' in action_dict.keys():
        for k in action_dict['outputs'].keys():
            append_to[action_name]['outputs'].append(k)
    return append_to


def get_action_inputs_and_outputs(dirnames):
    actions_with_ios = {}
    for a in dirnames:
        s = get_action_yaml(a)
        y = yaml.safe_load(s)  # pylint: disable=no-member
        actions_with_ios = append_action_inputs_and_outputs(
            actions_with_ios, a, y)
    return actions_with_ios


def get_errors_from_action(a, action_inputs_and_outputs):
    errors = []
    s = get_action_yaml(a)
    y = yaml.safe_load(s)  # pylint: disable=no-member

    # VALIDATION: action must have 'name' and 'description' fields
    errors = append_error_if_dict_key_missing(
        y, ['name', 'description'], errors)
    # VALIDATION: inputs and output must be lowercase alphanumeric with hyphens
    if 'inputs' in y.keys():
        errors = append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(
            y['inputs'], errors, " in 'inputs'")
    if 'outputs' in y.keys():
        errors = append_error_if_dict_keys_not_lowercase_alphanumeric_with_hyphens(
            y['outputs'], errors, " in 'outputs'")

    # VALIDATION: inputs and outputs must have 'description' field
    if 'inputs' in y.keys():
        for i in y['inputs'].keys():
            errors = append_error_if_dict_key_missing(
                y['inputs'][i], ['description'], errors, f" in 'inputs.{i}'")
    if 'outputs' in y.keys():
        for o in y['outputs'].keys():
            errors = append_error_if_dict_key_missing(
                y['outputs'][o], ['description'], errors, f" in 'outputs.{o}'")

    # VALIDATION: vars, secrets and env vars must be uppercase ALPHANUMERIC with underscores
    errors = append_error_if_repo_var_not_uppercase_alphanumeric_with_underscore(
        s, ['env', 'var', 'secrets'], errors)

    # VALIDATION: check if all called inputs exist
    input_names = y['inputs'].keys() if 'inputs' in y.keys() else []
    errors = append_error_if_var_not_in_list(
        s, 'inputs', list(input_names), errors)

    # VALIDATION: 'uses' must refer to existing action
    if 'runs' in y.keys():
        errors = append_error_if_dict_steps_refer_nonexisting_local_action(
            y['runs'], action_inputs_and_outputs, errors)

    return errors


def get_errors_from_action_filenames(dirnames):
    errors = []
    for action_dir in dirnames:
        # VALIDATION: action should have action.yml (and not .yaml)
        action_yml = os.path.join(
            os.environ['DOT_GITHUB_PATH'], 'actions', action_dir, 'action.yml')
        invalid_action_yaml = os.path.join(
            os.environ['DOT_GITHUB_PATH'], 'actions', action_dir, 'action.yaml')
        action_yml_exists = os.path.isfile(action_yml)
        invalid_action_yaml_exists = os.path.isfile(invalid_action_yaml)
        if not action_yml_exists and invalid_action_yaml_exists:
            errors.append(
                f"invalid extension - expected {action_yml} but found {invalid_action_yaml}".format(action_yml, invalid_action_yaml))
        if not action_yml_exists and not invalid_action_yaml_exists:
            errors.append(
                f"missing or invalid action file - {action_yml} (nor invalid action.yaml) not found".format(action_yml))
        if action_yml_exists and invalid_action_yaml_exists:
            errors.append(f"duplicated action file - found both valid {action_yml} and invalid {invalid_action_yaml}")

        # VALIDATION: action dir name should have lower case alphanumeric and hyphen only
        if not is_lowercase_alphanumeric_with_hyphens(action_dir):
            errors.append(
                f"invalid action dir name '{action_dir}' - should be lowercase alphanumeric with hyphens")

    return errors


def get_errors_from_workflow_filenames(filenames):
    errors = []
    for f in filenames:
        # VALIDATION: .yml extension
        if not f.endswith('.yml'):
            errors.append(f"workflow {f} should end with .yml")

        # VALIDATION: should not contain underscore
        if '_' in f:
            errors.append(
                f"workflow {f} filename should not contain underscore - use hyphens")

        # VALIDATION: workflow filename can start with underscore and then it should be lower case alphanumeric and hyphen only
        if not re.match(r'^[_]{0,1}[a-z0-9][a-z0-9\-]+\.yml$', f):
            errors.append(
                f"invalid workflow filename '{f}' - should be lower alphanumeric with hyphen, optionally starting with underscore when it is sub-workflow, and ending with .yml")

    return errors


def main():
    exit_if_invalid_env_vars([])
    exit_if_invalid_path()

    action_dirnames = get_action_dirnames()
    action_inputs_and_outputs = get_action_inputs_and_outputs(action_dirnames)
    workflow_filenames = get_workflow_filenames()
    print_info('Found action dirnames: ' + ', '.join(action_dirnames))
    print_info('Found workflow filenames: ' + ', '.join(workflow_filenames))

    # Validate action dir name and file inside
    errors = get_errors_from_action_filenames(action_dirnames)
    if len(errors) > 0:
        for err in errors:
            print_warning(f"action filenames -> {err}")

    # Validate workflow file names
    errors = get_errors_from_workflow_filenames(workflow_filenames)
    if len(errors) > 0:
        for err in errors:
            print_warning(f"workflow filenames -> {err}")

    # Loop through actions and validate them
    for a in action_dirnames:
        action_errors = get_errors_from_action(a, action_inputs_and_outputs)
        if len(action_errors) > 0:
            for err in action_errors:
                print_warning(f"action {a} -> {err}")

    # Loop through workflows and validate them
    for w in workflow_filenames:
        workflow_errors = get_errors_from_workflow(w, action_inputs_and_outputs)
        if len(workflow_errors) > 0:
            for err in workflow_errors:
                print_warning(f"workflow {w} -> {err}")

    # TODO: check if any id is not duplicated
    # TODO: check if any _id is not duplicated (like job_id, not the field)
    # TODO: validate field in 'env' blocks
    # TODO: use ShellCheck to check bash blocks (replace with random string if necessary) or at least -n?
    # TODO: check number of maximum lines


if __name__ == "__main__":
    main()
