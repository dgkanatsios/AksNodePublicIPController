# Go parameters
GOCMD=go
GOBUILD=CGO_ENABLED=0 GOOS=linux $(GOCMD) build -a -installsuffix cgo
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
VERSION=0.2.10
REGISTRY ?= docker.io/dgkanatsios
PROJECT_NAME=aksnodepublicipcontroller

TAG?=$(shell git rev-list HEAD --max-count=1 --abbrev-commit)

buildlocal:
		$(GOBUILD)  -o ./bin/app .
deps:
		dep ensure
buildremote: clean test
		docker build -f ./Dockerfile -t $(REGISTRY)/$(PROJECT_NAME):$(VERSION) .
		docker tag $(REGISTRY)/$(PROJECT_NAME):$(VERSION) $(REGISTRY)/$(PROJECT_NAME):latest
		docker system prune -f
pushremote:
		docker push $(REGISTRY)/$(PROJECT_NAME):$(VERSION)
		docker push $(REGISTRY)/$(PROJECT_NAME):latest
buildremotedev: clean test
		docker build -f ./Dockerfile -t $(REGISTRY)/$(PROJECT_NAME):$(TAG) .
		docker system prune -f
pushremotedev:
		docker push $(REGISTRY)/$(PROJECT_NAME):$(TAG)
gofmt:
		go fmt ./...
test:
		golangci-lint run --config ./golangci.yml
		$(GOTEST) -v ./...
clean: 
		$(GOCLEAN)
		rm -f ./bin/app