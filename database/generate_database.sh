#!/bin/bash
set -e
sqlite3 gen.sqlite3 < generate_schema.sql
xo file:gen.sqlite3?loc=auto --template-path ../db_templates/ -o ../models/
rm gen.sqlite3