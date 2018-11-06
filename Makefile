COMMIT=$(shell git rev-parse --short HEAD)$(shell [[ $$(git status --porcelain --ignored) = "" ]] && echo -clean || echo -dirty)
TAG ?= $(shell git rev-parse --short HEAD)
BACKUP_IMAGE ?= quay.io/openshift-on-azure/backup:$(TAG)

# all is the default target to build everything
all: clean build  backup

build: generate
	go build ./...

clean:
	rm -f  coverage.outbackup

test: unit e2e

generate:
	go generate ./...

backup: generate
	go build -ldflags "-X main.gitCommit=$(COMMIT)" ./cmd/backup

backup-image: backup
	go get github.com/openshift/imagebuilder/cmd/imagebuilder
	imagebuilder -f Dockerfile.backup -t $(BACKUP_IMAGE) .

backup-push: backup-image
	docker push $(BACKUP_IMAGE)

verify:
	./hack/validate-generated.sh
	go vet ./...
	./hack/verify-code-format.sh

unit: generate
	go test ./... -coverprofile=coverage.out
ifneq ($(ARTIFACT_DIR),)
	mkdir -p $(ARTIFACT_DIR)
	cp coverage.out $(ARTIFACT_DIR)
endif

cover: unit
	go tool cover -html=coverage.out

.PHONY: clean verify unit
