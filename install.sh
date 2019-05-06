#!/usr/bin/env bash
set -e
# Prompt for config to start with
echo "Enter the domain that you want to serve email for (e.g. mfashby.net)"
read DOMAIN

# Download the binary, make it executable and allow binding low ports
wget -O /usr/local/bin/henrymail https://github.com/MFAshby/henrymail/releases/download/0.0.1/henrymail
chmod +x /usr/local/bin/henrymail
setcap 'cap_net_bind_service=+ep' /usr/local/bin/henrymail

# Create a user for running henrymail
useradd -r henrymail

# Create a data directory for storing the database and set ownership
mkdir -p /var/lib/henrymail
chown henrymail:henrymail /var/lib/henrymail

# Download the sample config file and set the domain
mkdir -p /etc/henrymail
wget -O - https://raw.githubusercontent.com/MFAshby/henrymail/master/henrymail.sample.prop \
 | sed s/example.com/${DOMAIN}/ - > /etc/henrymail/henrymail.prop

# Download the systemd service definition and start
wget -O /etc/systemd/system/henrymail.service https://github.com/MFAshby/henrymail/raw/master/henrymail.service
systemctl daemon-reload
systemctl enable henrymail
systemctl start henrymail

# Download the uninstall script
wget https://github.com/MFAshby/henrymail/raw/master/uninstall.sh
chmod +x uninstall.sh

# Grep the admin password out of the logs
sleep 5
journalctl -fu henrymail | grep -qe Generated