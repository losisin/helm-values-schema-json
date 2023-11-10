#!/usr/bin/env sh

set -e

if ! 'helm plugin list | grep -q "schema"' > /dev/null 2>&1; then
    echo "Please install helm-values-schema-json plugin! https://github.com/losisin/helm-values-schema-json#install"
fi

helm schema "${@}"
