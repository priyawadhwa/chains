# Copyright 2021 The Tekton Authors

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


OPENAPIDEPS = openapi.yaml $(shell find pkg/types -iname "*.json")

TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)

# Binaries
SWAGGER := $(TOOLS_BIN_DIR)/swagger

## --------------------------------------
## Tooling Binaries
## --------------------------------------


$(GENSRC): $(SWAGGER) $(OPENAPIDEPS)
	$(SWAGGER) generate client -f openapi.yaml -q -r COPYRIGHT.txt -t pkg/generated --default-consumes application/json\;q=1
	$(SWAGGER) generate server -f openapi.yaml -q -r COPYRIGHT.txt -t pkg/generated --exclude-main -A rekor_server --exclude-spec --flag-strategy=pflag --default-produces application/json

.PHONY: validate-openapi
validate-openapi: $(SWAGGER)
	$(SWAGGER) validate openapi.yaml

$(SWAGGER): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -tags=tools -o $(TOOLS_BIN_DIR)/swagger github.com/go-swagger/go-swagger/cmd/swagger