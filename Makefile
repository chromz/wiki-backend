SHELL := /bin/bash

BACKEND := cmd/wiki/wiki.go
MDPROC := cmd/mdproc/mdproc.go
.PHONY: all

all:
	@go build $(BACKEND)


.PHONY: mdproc
mdproc:
	@go build $(MDPROC)
