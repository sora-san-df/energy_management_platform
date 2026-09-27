.PHONY: install test cover api web build run docker-build docker-run clean

install:
	go mod download
	cd web && npm ci

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

api:
	go run ./cmd/server

web:
	cd web && npm run dev

build:
	cd web && npm run build
	go build -o bin/server ./cmd/server

run: build
	./bin/server

docker-build:
	docker build -t voltia-energy .

docker-run:
	docker run --rm -p 8080:8080 --env-file .env -e APP_ENV=development -e DATABASE_PATH=/app/var/energy.db -e DATA_DIR=/app/data -e STATIC_DIR=/app/web/dist voltia-energy

clean:
	rm -rf bin web/dist energy.db energy.db-* coverage.out
