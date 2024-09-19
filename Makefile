lint:
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0 run --fix
