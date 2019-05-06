#!/usr/bin/env bash
set -e
rm -rf /etc/henrymail /var/lib/henrymail /usr/local/bin/henrymail /etc/systemd/system/henrymail.service
userdel henrymail