CONTAINER = byzcoin
IMAGE_NAME = c4dt/$(CONTAINER)
VERSION = $(shell git -C upstream/cothority fetch --tags; \
	git -C upstream/cothority tag | sort | tail -n 1 )
TAG = $(VERSION)-$(shell date +%y%m%d)
DOCKER_NAME = $(IMAGE_NAME)
DOW = $(shell date +%a)

# -s -w are for smaller binaries
# -X compiles the git tag into the binary
ldflags=-s -w -X main.gitTag=c4dt-$(TAG)

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

update: upstream-update pkg-update

test:
	cd cmd/byzcoin && ./test.sh -b

docker/byzcoin: cmd/byzcoin/main.go $(shell find pkg)
	docker run --rm -v "$$PWD":/usr/src/myapp -v "$$( go env GOPATH )":/go \
		-w /usr/src/myapp/cmd/byzcoin golang:1.13 go build -v
	cp cmd/byzcoin/byzcoin docker

docker/built: docker/byzcoin.sh docker/Dockerfile docker/byzcoin
	touch docker/built

docker: docker/built
	docker build -t $(DOCKER_NAME):$(TAG) docker
	docker tag $(DOCKER_NAME):$(TAG) $(DOCKER_NAME):$(DOW)
	docker tag $(DOCKER_NAME):$(TAG) $(DOCKER_NAME):latest

docker-push: docker
	docker push $(DOCKER_NAME):latest
	docker push $(DOCKER_NAME):$(DOW)
