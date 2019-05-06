#!/usr/bin/env bash
set -e
systemctl stop henrymail
systemctl disable henrymail
rm -rf /etc/henrymail /var/lib/henrymail /usr/local/bin/henrymail /etc/systemd/system/henrymail.service
userdel henrymail
systemctl daemon-reload