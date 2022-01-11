INC_DIR=../../rpc_proto
BASE_DIR=./pb

all: $(BASE_DIR)/common $(BASE_DIR)/user $(BASE_DIR)/node $(BASE_DIR)/container 

$(BASE_DIR)/common: $(INC_DIR)/common.proto
	mkdir -p $@
	protoc -I $(INC_DIR) --go_out=$@ --go_opt=paths=source_relative $^

$(BASE_DIR)/user: $(INC_DIR)/user_service.proto
	mkdir -p $@
	protoc -I $(INC_DIR) --go_out=$@ --go_opt=paths=source_relative \
		--go-grpc_out=$@ --go-grpc_opt=paths=source_relative $^

$(BASE_DIR)/node: $(INC_DIR)/node_service.proto $(INC_DIR)/node_types.proto
	mkdir -p $@
	protoc -I $(INC_DIR) --go_out=$@ --go_opt=paths=source_relative \
		--go-grpc_out=$@ --go-grpc_opt=paths=source_relative $^

$(BASE_DIR)/container: $(INC_DIR)/container_service.proto $(INC_DIR)/container_types.proto
	mkdir -p $@
	protoc -I $(INC_DIR) --go_out=$@ --go_opt=paths=source_relative \
		--go-grpc_out=$@ --go-grpc_opt=paths=source_relative $^
