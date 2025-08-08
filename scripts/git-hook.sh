#!/usr/bin/env sh

set -e

if ! (helm plugin list | grep -q "^schema\\s" > /dev/null 2>&1); then
    echo "Please install helm-values-schema-json plugin! https://github.com/losisin/helm-values-schema-json#install"
    exit 1
fi

helm schema "${@}"
