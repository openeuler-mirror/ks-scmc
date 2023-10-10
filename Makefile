PROJECT = ks-scmc
VERSION = 1.0.2
TAR_FILE = $(PROJECT)-$(VERSION).tar.gz
BINARY  = ks-scmc-server ks-scmc-authz ks-scmc-user
RPC     = rpc/pb
DESTDIR :=
PREFIX  := /usr
VAR     := /var
ETC     := /etc

all: $(BINARY)

$(BINARY): $(RPC)
	go build -ldflags="-s -w" scmc/cmd/$@

.PHONY: $(RPC)
$(RPC):
	cd rpc/ && make

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: env
env:
	go env -w GO111MODULE=on
	go env -w GOPROXY=https://goproxy.cn
	go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

.PHONY: clean
clean:
	rm -f $(BINARY)

.PHONY: install
install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m 755 -D $(BINARY) $(DESTDIR)$(PREFIX)/bin
	cd scripts/ && make install
	cd cmd/ks-scmc-user && make install

stat:
	cloc --fullpath  --not-match-d="(vendor|rpc/pb)" .

rpm: clean
	tar -zcf ~/rpmbuild/SOURCES/$(TAR_FILE) .
	cp $(PROJECT).spec ~/rpmbuild/SPECS/
	cd ~/rpmbuild/SPECS/
	rpmbuild -ba $(PROJECT).spec
