.PHONY: generate lint test check

SWAGGER := veniceai-api-docs/swagger.yaml
PATCHED := .swagger-patched.yaml

# Generate API client and types from the Venice OpenAPI spec.
# Applies swagger.patch to resolve duplicate type names before generation.
generate:
	patch -p1 $(SWAGGER) swagger.patch --output=$(PATCHED) || (rm -f $(PATCHED) && exit 1)
	go tool oapi-codegen --config oapi-codegen.yaml $(PATCHED)
	@rm -f $(PATCHED)

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

check: generate lint test
