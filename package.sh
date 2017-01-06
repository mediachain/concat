#!/bin/bash

OS_NAME=$(go env GOOS)
ARCH_NAME=$(go env GOARCH)

VERSION_STR=""
LAST_TAG=$(git describe --tags 2>/dev/null)
if [[ "${LAST_TAG}" == v* ]]; then
  VERSION_STR="-${LAST_TAG}"
fi

function make_tarball {
 prog=$1
 bin_dir=$2
 suffix=$3
 outfile="${PWD}/dist/${prog}${VERSION_STR}-${OS_NAME}-${ARCH_NAME}${suffix}.tgz"
 echo "creating package: ${outfile}"
 tar czf "${outfile}" -C "${bin_dir}" "${prog}"
}

mkdir -p ${PWD}/dist

if [ "$OS_NAME" == "linux" ]; then
  strip -o /tmp/mcnode ${GOPATH}/bin/mcnode
  strip -o /tmp/mcdir ${GOPATH}/bin/mcdir
  strip -o /tmp/mcid ${GOPATH}/bin/mcid
  make_tarball mcnode /tmp
  make_tarball mcdir /tmp
  make_tarball mcid /tmp
  make_tarball mcnode ${GOPATH}/bin -unstripped
  make_tarball mcdir ${GOPATH}/bin -unstripped
  make_tarball mcid ${GOPATH}/bin -unstripped
else
  make_tarball mcnode ${GOPATH}/bin
  make_tarball mcdir ${GOPATH}/bin
  make_tarball mcid ${GOPATH}/bin
fi
