.PHONY: centos-builder
centos-builder:
	docker build -f Dockerfile.centos8-builder -t nikos-centos8-builder .

.PHONY: nikos-centos8
nikos-centos8:
	docker run --rm -ti -v `pwd`:/go/src/github.com/lebauce/nikos nikos-centos8-builder go build -o nikos-centos8

.PHONY: nikos-dnf
nikos-dnf:
	PKG_CONFIG_PATH=/opt/nikos/embedded/lib/pkgconfig CGO_LDFLAGS="-Wl,-rpath,/opt/nikos/embedded/lib" go build -tags dnf

rpmdb:
	/opt/nikos/embedded/bin/rpmdb --initdb --root=/opt/nikos/embedded

test: nikos-nodnf
	cd tests/nikos; \
	molecule create

test.cos:
	docker run --rm --name focal -ti -v `pwd`/fixtures/cos/os-release:/etc/os-release:ro -v /opt/nikos:/opt/nikos -v `pwd`:/nikos ubuntu:focal bash -c "apt update; apt install -y ca-certificates; /nikos/nikos.sh download --verbose"

test.rhel:
	docker run --rm --name focal -ti -v `pwd`/fixtures/rhel/rpm:/var/lib/rpm:ro -v `pwd`/fixtures/rhel/os-release:/etc/os-release:ro -v `pwd`/fixtures/rhel/yum.repos.d:/etc/yum.repos.d:ro -v /opt:/opt -v `pwd`:/nikos -v `pwd`/fixtures/rhel/pki:/etc/pki:ro -v `pwd`/fixtures/rhel/rhsm:/etc/rhsm:ro ubuntu:focal /nikos/nikos.sh download --kernel 3.10.0-1160.2.1.el7.x86_64 --distribution RHEL
