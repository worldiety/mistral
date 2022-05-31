doc:
#	./gradlew doc
#	cp build/doc/index.adoc ./readme.adoc
	npx redoc-cli build --lint true -o ../../docs/redoc/index.html openapi.yaml
	npx @redocly/openapi-cli -o ../../docs/swagger/mistral.json bundle openapi.yaml

lint:
	npx @redocly/openapi-cli --lint=true -o build/lint.json bundle openapi.yaml

serve:
	npx @redocly/openapi-cli preview-docs openapi.yaml

build: lint doc

.DEFAULT_GOAL := build
