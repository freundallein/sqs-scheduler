
export BIN_DIR=/Users/freund/Documents/scheduler/bin
export CFG_PATH=/Users/freund/Documents/scheduler/backend/config.yml
export DOCKE_HUB_TAG=freundallein
# repository
init:
	echo "init"
fmt:
	cd backend && go fmt ./...

# dev
local:
	cd backend && go run cmd/local/main.go
test:
	cd backend && go run cmd/test/test.go
submitter:
	cd backend && go run cmd/submitter/submitter.go
worker:
	cd backend && go run cmd/worker/worker.go
scheduler:
	cd backend && go run cmd/scheduler/scheduler.go
resulter:
	cd backend && go run cmd/resulter/resulter.go
supervisor:
	cd backend && go run cmd/supervisor/supervisor.go
build:
	cd backend/cmd/submitter && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -o $$BIN_DIR/submitter
dockerbuild:
	docker build -t $$DOCKE_HUB_TAG/submitter -f backend/submitter/Dockerfile .
	docker build -t $$DOCKE_HUB_TAG/scheduler -f backend/scheduler/Dockerfile .
	docker build -t $$DOCKE_HUB_TAG/worker -f backend/worker/Dockerfile .
	docker build -t $$DOCKE_HUB_TAG/resulter -f backend/resulter/Dockerfile .
	docker build -t $$DOCKE_HUB_TAG/supervisor -f backend/supervisor/Dockerfile .
# Infrastructure
terraform:
	cd infrastructure/terraform && tf apply
up:
	docker-compose -f infrastructure/docker/docker-compose.yml up -d 
down:
	docker-compose -f infrastructure/docker/docker-compose.yml down