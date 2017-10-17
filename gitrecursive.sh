#!/usr/bin/env bash

home=$PWD
FC=$'\033[30m'
SC=$'\033[33m'
NC=$'\033[0m'


set -ue

repositories_path=$(find $PWD -type directory -name '.git')

function exec_command() {
	local prefix=$1
	local command=$2
	result=$( git ${command} ) #| sed "s#^#${prefix} #")
	echo -e ${prefix}$'\n'${result}
	echo
}

for repo_git_path in $repositories_path; do
    repo_path=${repo_git_path%/*}
    repo_name=${repo_path##*/}
    repo_dir=${repo_path%/*}
    cd $repo_path
    colored_path="${FC}${repo_dir}/${SC}${repo_name}${NC}"
    exec_command ${colored_path} "$@" &
    cd $home
done

wait < <(jobs -p)
