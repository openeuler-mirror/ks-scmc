PATH := $(shell go env GOPATH)/bin:$(PATH)

INC := ../rpc_proto
BASE := pb

GRPC_SERVICES := $(patsubst $(INC)/%,%, $(wildcard $(INC)/*.proto))
GRPC_SERVICES := $(patsubst %.proto,%, $(GRPC_SERVICES))
GRPC_SERVICES := $(patsubst %,$(BASE)/%, $(GRPC_SERVICES))

all: $(GRPC_SERVICES)

$(BASE)/%: $(INC)/%.proto
	mkdir -p $@
	protoc -I $(INC) --go_out=$@ --go_opt=paths=source_relative \
		--go-grpc_out=$@ --go-grpc_opt=paths=source_relative $(notdir $^)
clean:
	rm -rf $(BASE)
