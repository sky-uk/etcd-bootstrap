pkgs := $(shell go list ./... | grep -v /vendor/)
files := $(shell find . -path ./vendor -prune -o -name '*.go' -print)
image := skycirrus/etcd-bootstrap

git_rev := $(shell git rev-parse --short HEAD)
git_tag := $(shell git tag --points-at=$(git_rev))
release_date := $(shell date +%d-%m-%Y)
latest_git_tag := $(shell git for-each-ref --format="%(tag)" --sort=-taggerdate refs/tags | head -1)
latest_git_rev := $(shell git rev-list --abbrev-commit -n 1 $(latest_git_tag))
version := $(if $(git_tag),$(git_tag),dev-$(git_rev))
build_time := $(shell date -u)
ldflags := -X "github.com/sky-uk/etcd-bootstrap/cmd.version=$(version)" -X "github.com/sky-uk/etcd-bootstrap/cmd.buildTime=$(build_time)"

.PHONY: all format test build setup vet lint check-format check docker release

all : format check install
check : vet lint test
travis : setup check-format check build docker

setup:
	@echo "== setup"
	go get -v golang.org/x/lint/golint
	go get golang.org/x/tools/cmd/goimports

format :
	@echo "== format"
	@goimports -w $(files)
	@sync

build :
	@echo "== build"
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s $(ldflags)' -v

install :
	@echo "== install"
	@go install -ldflags '$(ldflags)' -v

unformatted = $(shell goimports -l $(files))

check-format :
	@echo "== check formatting"
ifneq "$(unformatted)" ""
	@echo "needs formatting:"
	@echo "$(unformatted)" | tr ' ' '\n'
	$(error run 'make format')
endif

vet :
	@echo "== vet"
	@go vet $(pkgs)

lint :
	@echo "== lint"
	@for pkg in $(pkgs); do \
		golint -set_exit_status $$pkg || exit 1 ; \
	done;

test :
	@echo "== run tests"
	@go test -race $(pkgs)

docker : build
	@echo "== build docker image"
	docker build -t $(image):latest .

release : docker
ifeq ($(strip $(git_tag)),)
	@echo "no tag on $(git_rev), skipping docker release"
else
	@echo "== release docker"
	@echo "releasing $(image):$(git_tag)"
	@docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD)
	docker tag $(image):latest $(image):$(git_tag)
	docker push $(image):$(git_tag)
	@if [ "$(git_rev)" = "$(latest_git_rev)" ]; then \
		echo "updating latest image"; \
		echo docker push $(image):latest ; \
	fi;
endif
