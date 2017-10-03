#!/usr/bin/env bash

home=$PWD

repositories_path=$(find $PWD -type directory -name '.git')

for repo_git_path in $repositories_path; do
    repo_path=${repo_git_path%/*}
    repo_name=${repo_path##*/}
    repo_dir=${repo_path%/*}
    cd $repo_path
    echo -e "\033[30m${repo_dir}/\033[33m${repo_name}\033[0m"
    git $@
    cd $home
done

