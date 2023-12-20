#!/usr/bin/env bash

set -e

tmp=$(mktemp -d manywest.XXXXX)

if [ -z "${tmp+x}" ] || [ -z "$tmp" ]; then
    echo "Error: \$tmp is not set or is an empty string."
    exit 1
fi

{
    rg --files . \
        | grep -v $tmp/filelist.txt \
        | grep -vE 'manywest$' \
        | grep -v README.org \
        | grep -v make_txtar.sh \
        | grep -v go.sum \
        | grep -v go.mod \
        | grep -v Makefile \
        | grep -v cmd/main.go \
        | grep -v logger.go \
        # | grep -v manywest.go \

} | tee $tmp/filelist.txt
tar -cf $tmp/manywest.tar -T $tmp/filelist.txt
mkdir -p $tmp/manywest
tar xf $tmp/manywest.tar -C $tmp/manywest
rg --files $tmp/manywest
txtar-c $tmp/manywest | pbcopy

rm -rf $tmp
