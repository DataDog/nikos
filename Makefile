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
	rpmdb --initdb --root=/opt/nikos/embedded

test: nikos-nodnf
	cd tests/nikos; \
	molecule create
