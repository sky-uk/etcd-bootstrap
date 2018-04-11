pkgs := $(shell go list ./... | grep -v /vendor/)
files := $(shell find . -path ./vendor -prune -o -name '*.go' -print)

.PHONY: all format test build vet lint checkformat check docker release

all : format check install
check : vet lint test
travis : checkformat check build docker

format :
	@echo "== format"
	@goimports -w $(files)
	@sync

build :
	@echo "== build"
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v

install :
	@echo "== install"
	@go install -v

unformatted = $(shell goimports -l $(files))

checkformat :
	@echo "== check formatting"
ifneq "$(unformatted)" ""
	@echo "needs formatting: $(unformatted)"
	@echo "run make format"
	@exit 1
endif

vet :
	@echo "== vet"
	@go vet $(pkgs)

lint :
	@echo "== lint"
	@for pkg in $(pkgs); do \
		golint -set_exit_status $$pkg || exit 1; \
	done;

test :
	@echo "== run tests"
	@go test -race $(pkgs)

git_rev := $(shell git rev-parse --short HEAD)
git_tag := $(shell git tag --points-at=$(git_rev))
etcd_version := v2.3.8
cloud_image := skycirrus/cloud-etcd-$(etcd_version)
vmware_image := skycirrus/vmware-etcd-$(etcd_version)

docker : build
	@echo "== build cloud"
	cp etcd-bootstrap cloud-etcd/
	docker build -t $(cloud_image):latest cloud-etcd/
	rm -f cloud-etcd/etcd-bootstrap

	@echo "== build vmware"
	cp etcd-bootstrap vmware-etcd/
	docker build -t $(vmware_image):latest vmware-etcd
	rm -f vmware-etcd/etcd-bootstrap

release : docker
	@echo "== release"
ifeq ($(strip $(git_tag)),)
	@echo "no tag on $(git_rev), skipping release"
else
	@echo "releasing $(cloud_image):$(git_tag)"
	@docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD)
	docker tag $(cloud_image):latest $(cloud_image):$(git_tag)
	docker push $(cloud_image):$(git_tag)

	@echo "releasing $(vmware_image):$(git_tag)"
	@docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD)
	docker tag $(vmware_image):latest $(vmware_image):$(git_tag)
	docker push $(vmware_image):$(git_tag)
endif
