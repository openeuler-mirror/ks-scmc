INC := ../../rpc_proto
GRPC_PLUGIN := $(shell which grpc_cpp_plugin)

PROTO_TYPES := $(patsubst $(INC)/%,%, $(wildcard $(INC)/*.proto))
PROTO_TYPES := $(patsubst %.proto,%.pb.h, $(PROTO_TYPES))

GRPC_SERVICES := $(patsubst $(INC)/%,%, $(wildcard $(INC)/*service.proto))
GRPC_SERVICES := $(patsubst %.proto,%.grpc.pb.h, $(GRPC_SERVICES))

all: $(PROTO_TYPES) $(GRPC_SERVICES)

%.grpc.pb.h %.grpc.pb.cc: $(INC)/%.proto
	protoc -I $(INC) --grpc_out=. --plugin=protoc-gen-grpc=$(GRPC_PLUGIN) $(notdir $^)

%.pb.h %.pb.cc: $(INC)/%.proto
	protoc -I $(INC) --cpp_out=. $(notdir $^)

clean:
	rm -f *.cc *.h
