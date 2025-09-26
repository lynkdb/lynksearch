# Copyright 2025 Eryx <evorui at gmail dot com>, All rights reserved.
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


LYNKAPI_FITTER_CMD = lynkapi-fitter
LYNKAPI_FITTER_ARGS = pkg/lynksearch

PROTOC_CMD = protoc
API_PROTOC_ARGS = --proto_path=./api/lynksearch/ --go_opt=paths=source_relative --go_out=./pkg/lynksearch/ --go-grpc_out=./pkg/lynksearch/ ./api/lynksearch/lynksearch.proto


.PHONY: api

all:
	@echo ""
	@echo "build complete"
	@echo ""

api:
	$(PROTOC_CMD) $(API_PROTOC_ARGS)
	$(LYNKAPI_FITTER_CMD) $(LYNKAPI_FITTER_ARGS)

