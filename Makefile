SHELL := /bin/bash

BACKEND := cmd/wiki/wiki.go
.PHONY: all

all:
	@go build $(REPLICATOR_CLIENT)
