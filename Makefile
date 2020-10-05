.PHONY: centos-builder
centos-builder:
	docker build -f Dockerfile.centos8-builder -t igor-centos8-builder .

.PHONY: igor-centos8
igor-centos8:
	docker run --rm -ti -v `pwd`:/go/src/github.com/lebauce/igor igor-centos8-builder go build -o igor-centos8

.PHONY: igor-dnf
igor-dnf:
	PKG_CONFIG_PATH=/opt/igor/embedded/lib/pkgconfig CGO_LDFLAGS="-Wl,-rpath,/opt/igor/embedded/lib" go build -tags dnf

rpmdb:
	rpmdb --initdb --root=/opt/igor/embedded

test: igor-nodnf
	cd tests/igor; \
	molecule create
