#include "rpc.h"

#include <chrono>
#include <map>
#include <memory>
#include <mutex>
#include <thread>

#include <grpcpp/impl/codegen/client_interceptor.h>
#include <kiran-log/qt5-log-i.h>

static const int RPC_CONNECT_TIMEOUT = 1000; // milliseconds

static std::mutex                                            s_channels_protect;
static std::map<std::string, std::shared_ptr<grpc::Channel>> s_channels;

using grpc::experimental::ClientRpcInfo;
using HookPoints = grpc::experimental::InterceptionHookPoints;

class LoggingInterceptor : public grpc::experimental::Interceptor
{
public:
  LoggingInterceptor(grpc::experimental::ClientRpcInfo *info) { info_ = info; }

  virtual void Intercept(grpc::experimental::InterceptorBatchMethods *methods)
  {
    if (methods->QueryInterceptionHookPoint(HookPoints::PRE_SEND_MESSAGE))
    {
      m_begin = std::chrono::steady_clock::now();
      auto request = static_cast<const google::protobuf::Message *>(methods->GetSendMessage());
      // TODO DEBUG LOG
      KLOG_INFO() << "RPC " << info_->method() << "\nrequest: " << request->DebugString().data();
    }
    if (methods->QueryInterceptionHookPoint(HookPoints::POST_RECV_STATUS))
    {
      auto reply = static_cast<const google::protobuf::Message *>(methods->GetRecvMessage());
      auto status = methods->GetRecvStatus();
      auto time_cost = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::steady_clock::now() - m_begin).count();

      if (status->ok())
      {
        // TODO DEBUG LOG
        KLOG_INFO() << "RPC " << info_->method()
                    << "\ncost: " << time_cost << "[ms]"
                    << "\nreply: " << reply->DebugString().data();
      }
      else
      {
        KLOG_INFO() << "RPC " << info_->method()
                    << "\ncost: " << time_cost << "[ms]"
                    << "\nerror: " << status->error_code()
                    << " " << status->error_message().data();
      }
    }
    methods->Proceed();
  }

private:
  grpc::experimental::ClientRpcInfo *info_;
  std::chrono::steady_clock::time_point m_begin;
};

class LoggingInterceptorFactory
    : public grpc::experimental::ClientInterceptorFactoryInterface {
 public:
  virtual grpc::experimental::Interceptor* CreateClientInterceptor(
      grpc::experimental::ClientRpcInfo* info) override {
    return new LoggingInterceptor(info);
  }
};

static bool is_channel_connected(std::shared_ptr<grpc::Channel> &chan, int timeout) {
    return chan->WaitForConnected(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                gpr_time_from_millis(timeout, GPR_TIMESPAN)));
}

static std::shared_ptr<grpc::Channel> new_rpc_channel(const std::string& addr) {
    grpc::ChannelArguments args;
    std::vector<std::unique_ptr<grpc::experimental::ClientInterceptorFactoryInterface>> creators;
    creators.push_back(std::unique_ptr<LoggingInterceptorFactory>(new LoggingInterceptorFactory()));

    for (int i = 0; i < 2; i++) {
        auto chan = grpc::experimental::CreateCustomChannelWithInterceptors(
          addr, grpc::InsecureChannelCredentials(), args, std::move(creators));
        // auto chan = grpc::CreateChannel(addr, grpc::InsecureChannelCredentials());
        if (chan == nullptr || !is_channel_connected(chan, RPC_CONNECT_TIMEOUT)) {
            // wait for a while before retry.
            std::this_thread::sleep_for(std::chrono::milliseconds(500));
            continue;
        }
        
        // got a connected channel, return
        s_channels[addr] = chan;
        return chan;
    }
    return nullptr;
}

std::shared_ptr<grpc::Channel> get_rpc_channel(const std::string& addr) {
    std::lock_guard<std::mutex> guard(s_channels_protect);
    auto it = s_channels.find(addr);
    if (it != s_channels.end()) {
        auto chan = it->second;
        if (is_channel_connected(chan, RPC_CONNECT_TIMEOUT)) {
            return chan;
        }
    }

    // connection not exist or connection is dead, get a new one.
    return new_rpc_channel(addr);
}
