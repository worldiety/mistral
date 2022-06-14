doc:
#	./gradlew doc
#	cp build/doc/index.adoc ./readme.adoc
	npx redoc-cli build --lint true -o ../../docs/redoc/index.html openapi.yaml
	npx @redocly/openapi-cli -o ../../docs/swagger/mistral.json bundle openapi.yaml

lint:
	rm -rf build/lint.json
	npx @redocly/openapi-cli --lint=true -o build/lint.json bundle openapi.yaml

serve:
	npx @redocly/openapi-cli preview-docs openapi.yaml

build: lint doc

testGenMod:
	mkdir -p build/testmod
	cd build/testmod && go mod init testmod
	cd build/testmod && go get github.com/deepmap/oapi-codegen/pkg/runtime
	cd build/testmod && go get github.com/deepmap/oapi-codegen/pkg/types
	cd build/testmod && go get github.com/go-chi/chi/v5

# go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest
testGen: build
	sh testenv.sh
	oapi-codegen -config=oapi-config.yaml build/lint.json > build/testmod/mistral.gen.go
	cd build/testmod && go build ./...

.DEFAULT_GOAL := build
