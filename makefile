OS := $(shell uname -s)

build:
	@echo "Building on $(OS)"
	go mod tidy
	go build -o bin/migrate_user migration/migrate_user.go
	go build -o bin/user_service user-service/application.go