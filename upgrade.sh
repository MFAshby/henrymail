#!/usr/bin/env bash
set -e

systemctl stop henrymail
rm /usr/local/bin/henrymail
wget -O /usr/local/bin/henrymail https://github.com/MFAshby/henrymail/releases/latest/download/henrymail
chmod +x /usr/local/bin/henrymail
setcap 'cap_net_bind_service=+ep' /usr/local/bin/henrymail
systemctl start henrymail