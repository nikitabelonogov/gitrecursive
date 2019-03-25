#!/usr/bin/env bash

FC=$'\033[30m'
SC=$'\033[33m'
NC=$'\033[0m'


set -ue

function exec_command() {
  local prefix=$1
  local cmd=$2
  local stdout=$($cmd)
  echo -e "${prefix}\n${stdout}"
}

repositories_path=$(find ${PWD} -type directory -name '.git')

for repo_git_path in $repositories_path; do
  repo_path=${repo_git_path%/*}
  repo_name=${repo_path##*/}
  colored_path=${repo_path/${PWD}/${FC}${PWD}${NC}}
  colored_path=${colored_path/${repo_name}/${SC}${repo_name}${NC}}
  gitcmd=${@#*git}
  exec_command "${colored_path} ${gitcmd}" "git -C ${repo_path} ${gitcmd}" &
done

wait < <(jobs -p)
