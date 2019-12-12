upstream:
	mkdir upstream

upstream/cothority:
	git clone https://github.com/dedis/cothority --depth 1 upstream/cothority

upstream/onet:
	git clone https://github.com/dedis/onet --depth 1 upstream/onet

upstream-update: upstream upstream/cothority upstream/onet
	( cd upstream/cothority; git pull )
	( cd upstream/onet; git pull )

build-update:
	rm -rf build
	mkdir build
	for a in $$( cat build.files ); do \
		echo "Copying files $$a"; \
		d=build/$$( dirname $$a ); \
		mkdir -p $$d; \
		cp -a upstream/$$a $$d; \
	done
	cp -a build.base/ build

update: upstream-update build-update

test:

docker: docker-base docker-configure docker-byzcoin

docker-base: docker/Dockerfile-base
	docker build
