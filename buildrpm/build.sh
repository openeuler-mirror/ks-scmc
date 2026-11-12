#!/bin/bash

set -x

use_git_code=0
build_type="container" # local, mock, container
container_build_image="scmc-compiler:"
pkgname=ks-scmc

