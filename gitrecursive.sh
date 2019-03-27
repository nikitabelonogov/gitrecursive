#!/usr/bin/env bash

FC=$'\033[30m'
SC=$'\033[33m'
NC=$'\033[0m'


set -ue

for repo_git_path in $(find ${PWD} -type directory -name '.git'); do
  repo_path=${repo_git_path%/*}
  repo_name=${repo_path##*/}
  colored_path=${repo_path/${PWD}/${FC}${PWD}${NC}}
  colored_path=${colored_path/${repo_name}/${SC}${repo_name}${NC}}
  echo -e "${colored_path} $@\n$(git -C ${repo_path} $@)" &
done

wait < <(jobs -p)
