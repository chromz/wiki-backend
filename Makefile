SHELL := /bin/bash

BACKEND := cmd/wiki/wiki.go
IMGPROC := cmd/imgproc/imgproc.go
.PHONY: all

all:
	@go build $(BACKEND)


.PHONY: imgproc
imgproc:
	@go build $(IMGPROC)
