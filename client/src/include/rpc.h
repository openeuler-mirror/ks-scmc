#ifndef RPC_H_
#define RPC_H_

#include <grpcpp/grpcpp.h>

#include "container_service.grpc.pb.h"
#include "node_service.grpc.pb.h"
#include "image_service.grpc.pb.h"
#include "user_service.grpc.pb.h"
#include "network_service.grpc.pb.h"

/*
 * Example:
    auto chan = get_rpc_channel(address);
    if (m_Channel != nullptr) {
        NodeClient cli(m_Channel);
        cli.List(request);
    }
 */
std::shared_ptr<grpc::Channel> get_rpc_channel(const std::string& addr);

#endif
