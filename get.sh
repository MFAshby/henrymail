#!/usr/bin/env bash
set -e
wget https://raw.githubusercontent.com/MFAshby/henrymail/master/install.sh \
    && chmod +x install.sh \
    && sudo ./install.sh