#!/usr/bin/env bash

RUN_PWD="${PWD}"
GO_ORB="github.com/go-orb/"
GO_ORB_LEN="${#GO_ORB}"
GO_ORB_PLUGINS="${GO_ORB}plugins/"
GO_ORB_PLUGINS_LEN="${#GO_ORB_PLUGINS}"

pushd "${1}"; 
for package in $(grep --null -E "^replace\s+github.com/go-orb/[\/a-zA-Z\-]+ => .*$" go.mod); do
    if [[ "${package:0:$GO_ORB_LEN}" != "${GO_ORB}" ]]; then
        continue;
    fi

    # We do not remove "internal" replaces.
    if [[ "${package:0:$GO_ORB_PLUGINS_LEN}" == "${GO_ORB_PLUGINS}" ]]; then
        continue;
    fi

    echo go mod edit -dropreplace="${package}";
    go mod edit -dropreplace="${package}";
    # We should replace @main with @latest once we have releases.
    go get -u "${package}@main"
done;
popd >/dev/null;