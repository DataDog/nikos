.PHONY: centos-builder
centos-builder:
	docker build -f Dockerfile.centos8-builder -t nikos-centos8-builder .

.PHONY: nikos-centos8
nikos-centos8:
	docker run --rm -ti -v `pwd`:/go/src/github.com/DataDog/nikos nikos-centos8-builder go build -o nikos-centos8

test: nikos-nodnf
	cd tests/nikos; \
	molecule create

test.cos:
	docker run --rm --name focal -ti -v `pwd`/fixtures/cos/os-release:/etc/os-release:ro -v /opt/nikos:/opt/nikos -v `pwd`:/nikos ubuntu:focal bash -c "apt update; apt install -y ca-certificates; /nikos/nikos.sh download --verbose"

test.rhel:
	docker run --rm --name focal -e HOST_ETC=/host/etc -ti -v `pwd`/fixtures/rhel/rpm:/var/lib/rpm:ro -v `pwd`/fixtures/rhel/os-release:/etc/os-release:ro -v `pwd`/fixtures/rhel/yum.repos.d:/etc/yum.repos.d:ro -v /opt:/opt -v `pwd`:/nikos -v `pwd`/fixtures/rhel/pki:/etc/pki:ro -v `pwd`/fixtures/rhel/rhsm:/etc/rhsm:ro ubuntu:focal /nikos/nikos.sh download --kernel 3.10.0-1160.2.1.el7.x86_64 --distribution RHEL

test.ubuntu:
	docker run --rm --name focal -e HOST_ETC=/host/etc -ti -v ./fixtures/ubuntu/focal:/host:ro -v $(pwd)/..:/nikos ubuntu:focal /nikos/nikos.sh download --kernel $(cat /nikos/fixtures/ubuntu/focal/kernel) --arch $(cat /nikos/fixtures/ubuntu/focal/arch)'

test.centos:
	docker run -e HOST_ETC=/host/etc -v /opt:/opt:ro -v $$(pwd)/centos/7:/host:ro -v $$(dirname $$(pwd)):/nikos -ti fedora bash -c '/nikos/nikos -v download --kernel $$(cat /nikos/fixtures/centos/7/kernel) --arch $$(cat /nikos/fixtures/centos/7/arch)'
