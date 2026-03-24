#!/usr/bin/env bash

set -ue ${DEBUG:+-x}

FC=$'\033[30m'
SC=$'\033[33m'
NC=$'\033[0m'

depth_args=()
args=()
while [[ $# -gt 0 ]]; do
  case $1 in
    -d|--depth)
      depth_args=(-maxdepth "$(($2 + 1))")
      shift 2
      ;;
    *)
      args+=("$1")
      shift
      ;;
  esac
done
set -- "${args[@]+"${args[@]}"}"

for repo_git_path in $(find ${PWD} "${depth_args[@]+"${depth_args[@]}"}" -type directory -name '.git'); do
  repo_path=${repo_git_path%/*}
  repo_name=${repo_path##*/}
  colored_path=${repo_path/${PWD}/${FC}${PWD}${NC}}
  colored_path=${colored_path/${repo_name}/${SC}${repo_name}${NC}}
  echo "${colored_path}"
  git -C "${repo_path}" "$@" || true
  echo "---"
done

wait
