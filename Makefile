version := 0.0.12
.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "Makefile Commands:"
	@echo "----------------------------------------------------------------"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
	@echo "----------------------------------------------------------------"

.PHONY: proto
proto: ## regenerate gRPC code
	@echo "generating protobuf code..."
	@prototool generate
	@go fmt ./...

.PHONY: up
up: ## start docker containers
	@docker-compose -f docker-compose.yml pull
	@docker-compose -f docker-compose.yml up -d

.PHONY: down
down: ## shuts down docker containers
	docker-compose -f docker-compose.yml down --remove-orphans

run: ## run server
	@go run main.go

version: ## iterate sem-ver
	bumpversion patch --allow-dirty

tag: ## tag sem-ver
	git tag v$(version)

push: docker-build docker-push ## rebuild & push docker image then push updated code to github
	git push origin master
	git push origin v$(version)

docker-build: ## build docker image
	docker build -t colemanword/geodb:$(version) .

docker-push: ## push docker image
	docker push colemanword/geodb:$(version)

docker-run: ## run docker image
	docker run -d colemanword/geodb:$(version) -p 8080:8080

test: ## run tests
	@go test -v