#!/usr/bin/env bash

# TODO: run specific services.

cmd = $1
shift

case $cmd in
tools)
  sleep inifinity
  ;;
*)
  /userclouds/bin/$cmd $@
  ;;
esac
