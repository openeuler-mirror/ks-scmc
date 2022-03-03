PROJECT = ks-scmc
BINARY  = ks-scmc-server
RPC     = rpc/pb
DESTDIR :=
PREFIX  := /usr
VAR     := /var
ETC     := /etc

all: $(BINARY)

$(BINARY): $(RPC)
	go build scmc/cmd/$@

.PHONY: $(RPC)
$(RPC):
	cd rpc/ && make

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: env
env:
	go env -w GO111MODULE=on
	go env -w GOPROXY=https://goproxy.io,direct
	go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -rf $(RPC)

.PHONY: install
install:
	install -d $(DESTDIR)$(ETC)/$(PROJECT)/
	install -d $(DESTDIR)$(VAR)/lib/$(PROJECT)/containers
	install -d $(DESTDIR)$(VAR)/log/$(PROJECT)/
	install -d $(DESTDIR)$(PREFIX)/bin/
	install -m 755 $(BINARY) $(DESTDIR)$(PREFIX)/bin/
	cd scripts/ && make DESTDIR=$(DESTDIR) install