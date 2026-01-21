.PHONY: build-css generate run test warmup

build-css:
	npx tailwindcss -i ./internal/assets/css/input.css -o ./internal/assets/css/output.css

generate:
	go run github.com/a-h/templ/cmd/templ@latest generate

run: generate build-css
	go run cmd/server/main.go

test:
	go test ./...

warmup:
	go run cmd/warmup/main.go
