#!/usr/bin/env bash

# Stop the service
systemctl stop henrymail

# Restore the database and config files
unzip backup.zip -d /

# Restart the service
systemctl start henrymail