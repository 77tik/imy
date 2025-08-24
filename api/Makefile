override NAMING_FORMAT := go_zero
override ZERO_ARGS := --style $(NAMING_FORMAT)

ifndef ATTILA_DEBUG
override OUTPUT_REDIRECT := 2>&1 | grep -v 'ignored generation'
endif

override GO_ZERO_HOME := /tmp/.goctl

.PHONY: all
all: api swagger

api: --format --replace-template
	@echo "Generating go files for provided apis ..."
	@goctl api go --home $(GO_ZERO_HOME) --api main.api --dir ../ $(ZERO_ARGS) $(OUTPUT_REDIRECT)

swagger:
	@echo "Generating swagger files for provided apis ..."
	@goctl api plugin -plugin goctl-openapi -api main.api -dir ../internal/handler/swagger


# ----------------------------------------------------------
# Private targetscd
# ----------------------------------------------------------

--format:
	@echo "Format api files ..."
	@goctl api format --dir .

--generate-template:
	@echo "Initialize the templates ..."
	@rm -rf $(GO_ZERO_HOME)
	@goctl template init --home $(GO_ZERO_HOME)

--replace-template: --generate-template
	@echo "Replacing the templates ..."
	@$(eval HANDLER_TEMPLATE=$(shell find $(GO_ZERO_HOME) -name 'handler.tpl' | sort -ru | head -n 1))
	@cp ./handler_example.tpl $(HANDLER_TEMPLATE)
