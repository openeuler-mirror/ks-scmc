all: common user node container 

INC_DIR=../../rpc_proto
BASE_DIR=./pb

common: $(INC_DIR)/common.proto
	mkdir -p $(BASE_DIR)/$@
	protoc -I $(INC_DIR) --go_out=$(BASE_DIR)/$@ --go_opt=paths=source_relative $^

user: $(INC_DIR)/user_service.proto
	mkdir -p $(BASE_DIR)/$@
	protoc -I $(INC_DIR) --go_out=$(BASE_DIR)/$@ --go_opt=paths=source_relative \
		--go-grpc_out=$(BASE_DIR)/$@ --go-grpc_opt=paths=source_relative $^

node: $(INC_DIR)/node_service.proto $(INC_DIR)/node_types.proto
	mkdir -p $(BASE_DIR)/$@
	protoc -I $(INC_DIR) --go_out=$(BASE_DIR)/$@ --go_opt=paths=source_relative \
		--go-grpc_out=$(BASE_DIR)/$@ --go-grpc_opt=paths=source_relative $^

container: $(INC_DIR)/container_service.proto $(INC_DIR)/container_types.proto
	mkdir -p $(BASE_DIR)/$@
	protoc -I $(INC_DIR) --go_out=$(BASE_DIR)/$@ --go_opt=paths=source_relative \
		--go-grpc_out=$(BASE_DIR)/$@ --go-grpc_opt=paths=source_relative $^
