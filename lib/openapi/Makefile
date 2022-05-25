doc:
#	./gradlew doc
#	cp build/doc/index.adoc ./readme.adoc
	npx redoc-cli bundle --lint true -o ../../docs/index.html openapi.yaml

lint:
	npx @redocly/openapi-cli --lint=true -o build/lint.json bundle openapi.yaml

serve:
	npx @redocly/openapi-cli preview-docs openapi.yaml


.DEFAULT_GOAL := lint doc
