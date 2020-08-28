DOCKER_NAME := c4dt/byzcoin
NAME := c4dt
DOW := $(shell date +%a)

# On Monday, get today's date. On all other days of the week, get last
# Monday's date
ifeq '$(DOW)' 'Mon'
DATE_COMPILE := $(shell date +%Y%m%d)
else
# mac date doesn't know about --date argument...
DATE_COMPILE := $(shell date --date "last Monday" +%Y%m%d || \
               	date -v Mon +%Y%m%d)
endif

DOCKER_TAG = $(DATE_COMPILE)

LATEST_COMMIT = g$(shell git rev-parse --short HEAD)
BINARY_VERSION = $(NAME)-$(DATE_COMPILE)-$(LATEST_COMMIT)

# -s -w are for smaller binaries
# -X compiles the git tag into the binary
ldflags=-s -w -X main.gitTag=$(BINARY_VERSION)

print:
	echo $(DOW)
	echo $(DATE_COMPILE)

upstream:
	mkdir upstream

upstream/cothority:
	git clone https://github.com/dedis/cothority --depth 1 upstream/cothority
	cd upstream/cothority && \
	for d in $$(cat upstream_commits); do \
		git fetch origin $$d && \
		git cherry-pick $$d; \
	done

upstream/onet:
	git clone https://github.com/dedis/onet --depth 1 upstream/onet

upstream-update: upstream upstream/cothority upstream/onet
	( cd upstream/cothority; git pull )
	( cd upstream/onet; git pull )

.PHONY: pkg-clean pkg-update pkg-patch

pkg-clean:
	rm -rf pkg

pkg-update: pkg-clean
	mkdir -p pkg
	for a in $$( cat pkg.files ); do \
		echo "Copying files $$a"; \
		d=pkg/$$( dirname $$a ); \
		mkdir -p $$d; \
		cp -a `eval echo upstream/$$a` $$d; \
	done
	cp -av pkg.base/* pkg
	printf "\nreplace go.dedis.ch/onet/v3 => ../onet\n" >> pkg/cothority/go.mod

pkg-patch: BCA = pkg/cothority/byzcoin/bcadmin/main.go
pkg-patch:
	sed -i.bak '/v3.eventlog/d' $(BCA)
	sed -i.bak 's.v3/personhood.v3/personhood/contracts.' $(BCA)
	rm $(BCA).bak

.PHONY: update test

update: upstream-update pkg-update pkg-patch

test:
	cd cmd/byzcoin && ./test.sh -b

docker/byzcoin: cmd/byzcoin/main.go $(shell find pkg)
	docker run --rm -v "$$PWD":/usr/src/myapp \
		-v "$$PWD"/.godocker:/go \
		-w /usr/src/myapp golang:1.15 \
		sh -c "go build -ldflags='$(ldflags)' ./cmd/byzcoin; \
		cd pkg/cothority; go build -ldflags='$(ldflags)' ./byzcoin/bcadmin; \
		cd scmgr; go build -ldflags='$(ldflags)' ."
	mv pkg/cothority/bcadmin pkg/cothority/scmgr/scmgr $(@D)
	mv byzcoin $@

docker/built: docker/byzcoin.sh docker/Dockerfile docker/byzcoin
	touch $@

.PHONY: docker docker-push-new docker-push-dow docker-push-all

docker: docker/built
	docker build -t $(DOCKER_NAME):$(DOCKER_TAG) docker

docker-push-new: docker
	for t in $(DOCKER_TAG) latest Mon; do \
	  docker tag $(DOCKER_NAME):$(DOCKER_TAG) $(DOCKER_NAME):$$t; \
	  docker push $(DOCKER_NAME):$$t; \
	done

docker-push-dow:
	docker pull $(DOCKER_NAME):$(DOCKER_TAG)
	docker tag $(DOCKER_NAME):$(DOCKER_TAG) $(DOCKER_NAME):$(DOW)
	docker push $(DOCKER_NAME):$(DOW)

docker-push-all: DATE_COMPILE := $(shell date +%Y%m%d-%H%M)
docker-push-all: NAME := force
docker-push-all: docker
	docker push $(DOCKER_NAME):$(DOCKER_TAG)
	@for d in Sun Mon Tue Wed Thu Fri Sat; do \
		echo "Creating docker-image for $$d"; \
		docker tag $(DOCKER_NAME):$(DOCKER_TAG) $(DOCKER_NAME):$$d; \
		docker push $(DOCKER_NAME):$$d; \
	done
