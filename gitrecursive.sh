#!/usr/bin/env bash

FC=$'\033[30m'
SC=$'\033[33m'
NC=$'\033[0m'


set -ue

repositories_path=$(find ${PWD} -type directory -name '.git')

function exec_command() {
  local prefix=$1
  local cmd=$2
  echo -e "${prefix}\n$( ${cmd} )"
}

for repo_git_path in $repositories_path; do
  repo_path=${repo_git_path%/*}
  repo_name=${repo_path##*/}
  repo_dir=${repo_path%/*}
  colored_path="${FC}${repo_dir}/${SC}${repo_name}${NC}"
  cmd="git -C ${repo_path} ${@#*git}"
  exec_command ${colored_path} "$cmd" &
done

wait < <(jobs -p)
