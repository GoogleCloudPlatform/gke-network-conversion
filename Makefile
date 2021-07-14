# Copyright 2021 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOBIN ?= $(GOPATH)/bin
GONAME := gkeconvert
GOMAIN := main.go

check:
ifndef GOPATH
	@echo "GOPATH is empty"
	@false
endif

test: check
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go test -race -covermode=atomic ./...

build: check
	@echo "Building $(GOBIN)/$(GONAME)"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -race -v -o $(GOBIN)/$(GONAME) $(GOMAIN)

clean: check
	rm -f $(GOBIN)/$(GONAME)

all: check test build

.PHONY: check test build clean all