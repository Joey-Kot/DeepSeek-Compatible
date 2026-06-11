#!/bin/sh
# Copyright (C) 2026 Joey Kot <joey.kot.x@gmail.com>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

set -eu

if [ "$#" -eq 0 ] || [ "${1#-}" != "$1" ]; then
  if [ -n "${DEBUG_LOG_BODY:-}" ]; then
    set -- "--debug-log-body=${DEBUG_LOG_BODY}" "$@"
  fi
  if [ -n "${VERIFY_SSL:-}" ]; then
    set -- "--verify-ssl=${VERIFY_SSL}" "$@"
  fi
  if [ -n "${DEEPSEEK_HTTP_TIMEOUT:-}" ]; then
    set -- "--deepseek-http-timeout" "${DEEPSEEK_HTTP_TIMEOUT}" "$@"
  fi
  if [ -n "${DEEPSEEK_MODELS:-}" ]; then
    set -- "--deepseek-models" "${DEEPSEEK_MODELS}" "$@"
  fi
  if [ -n "${DEEPSEEK_MODEL:-}" ]; then
    set -- "--deepseek-model" "${DEEPSEEK_MODEL}" "$@"
  fi
  if [ -n "${DEEPSEEK_BASE_URL:-}" ]; then
    set -- "--deepseek-base-url" "${DEEPSEEK_BASE_URL}" "$@"
  fi
  if [ -n "${DEEPSEEK_API_KEY:-}" ]; then
    set -- "--deepseek-api-key" "${DEEPSEEK_API_KEY}" "$@"
  fi
  if [ -n "${API_TOKEN:-}" ]; then
    set -- "--api-token" "${API_TOKEN}" "$@"
  fi
  if [ -n "${LISTEN:-}" ]; then
    set -- "--listen" "${LISTEN}" "$@"
  fi

  set -- deepseek-compatible "$@"
fi

exec "$@"
