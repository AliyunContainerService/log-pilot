#!/bin/sh
#

pilot=$(ps aux | grep -v grep | grep pilot)

if [ -z "$pilot" ]; then
    exit 1
else
    exit 0
fi
