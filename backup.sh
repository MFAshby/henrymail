#!/usr/bin/env bash

# Stop the service so that files aren't changing
systemctl stop henrymail

# backup the database and config files
zip --encrypt -r backup.zip /etc/henrymail/* /var/lib/henrymail/*

# Restart the service
systemctl start henrymail