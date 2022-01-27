PROJECT = ksc-mcube
RPC = rpc/pb

all: $(PROJECT)

$(PROJECT): $(RPC)
	go build -o $@ $@/cmd

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
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONY: clean
clean:
	rm -f $(PROJECT)
	rm -rf $(RPC)
