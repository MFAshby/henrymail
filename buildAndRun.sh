#!/usr/bin/env bash
go generate ./... \
	&& go build \
	&& ./henrymail
