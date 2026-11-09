PLUGIN_NAME=hc-eps-payment-plugin
BINARY_NAME=hc-eps-payment-plugin

.PHONY: build test tidy ui

ui:
	npm install --prefix ui
	npm run build --prefix ui

build: ui
	go build -o $(BINARY_NAME) .

test:
	go test -count=1 ./...

tidy:
	go mod tidy
