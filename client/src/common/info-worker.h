#ifndef INFOWorker_H
#define INFOWorker_H

#include <QFuture>
#include <QFutureWatcher>
#include <QMutex>
#include <QObject>
#include <QPair>

#include <grpcpp/grpcpp.h>

#include "rpc.h"

extern std::string g_server_addr;

enum ErrCode
{
    ErrOK = 0,
    ErrCanceled = 1,            // "请求取消"
    ErrUnknown = 2,             // "未知错误"
    ErrInvalidArgument = 3,     // "参数错误"
    ErrDeadlineExceeded = 4,    // "请求超时"
    ErrNotFound = 5,            // "资源不存在"
    ErrAlreadyExists = 6,       // "资源冲突"
    ErrPermissionDenied = 7,    // "拒绝请求"
    ErrResourceExhausted = 8,   // ResourceExhausted framework error
    ErrFailedPrecondition = 9,  // FailedPrecondition
    ErrAborted = 10,            // Aborted
    ErrOutOfRange = 11,         // OutOfRange
    ErrUnimplemented = 12,      // Unimplemented framework error
    ErrInternal = 13,           // Internal, "内部错误" framework error | rpc failure
    ErrUnavailable = 14,        // Unavailable  // framework error
    ErrDataLoss = 15,           // DataLoss
    ErrUnauthenticated = 16,    // Unauthenticated, "用户请求未认证"
    ErrWrongPassword = 17,      // Errno_WrongPassword, "密码错误"

    ErrUserNotExist = -1,
    ErrStatus = -2,
};

class InfoWorker : public QObject
{
    Q_OBJECT
public:
    static InfoWorker& getInstance()
    {
        static InfoWorker s_InfoWorker;
        return s_InfoWorker;
    }

    // node management
    void listNode();
    void createNode(const node::CreateRequest &);
    void removeNode(const std::vector<int64_t> &node_ids);
    void nodeStatus(const std::vector<int64_t> &node_ids);

    // container management
    void listContainer(const std::vector<int64_t> &node_ids, const bool all);
    void createContainer(const container::CreateRequest &);
    void containerStatus(const int64_t node_id);
    void containerInspect(const int64_t node_id, const std::string &container_id);
    void startContainer(const std::map<int64_t, std::vector<std::string>> &ids);
    void stopContainer(const std::map<int64_t, std::vector<std::string>> &ids);
    void killContainer(const std::map<int64_t, std::vector<std::string>> &ids);
    void restartContainer(const std::map<int64_t, std::vector<std::string>> &ids);
    void removeContainer(const std::map<int64_t, std::vector<std::string>> &ids);
    void updateContainer(const container::UpdateRequest &);

    // network management
    void listNetwork(const int64_t);

    // image management
    void listImage(const int64_t);

private:
    InfoWorker(QObject* parent = nullptr);

    // node management
    static QPair<grpc::Status, node::ListReply> _listNode(const node::ListRequest &);
    static QPair<grpc::Status, node::CreateReply> _createNode(const node::CreateRequest &);
    static QPair<grpc::Status, node::RemoveReply> _removeNode(const node::RemoveRequest &);
    static QPair<grpc::Status, node::StatusReply> _nodeStatus(const node::StatusRequest &);

    // container management
    static QPair<grpc::Status, container::ListReply> _listContainer(const container::ListRequest &);
    static QPair<grpc::Status, container::CreateReply> _createContainer(const container::CreateRequest &);
    static QPair<grpc::Status, container::StatusReply> _containerStatus(const container::StatusRequest &);
    static QPair<grpc::Status, container::InspectReply> _containerInspect(const container::InspectRequest &);
    static QPair<grpc::Status, container::StartReply> _startContainer(const container::StartRequest &);
    static QPair<grpc::Status, container::StopReply> _stopContainer(const container::StopRequest &);
    static QPair<grpc::Status, container::KillReply> _killContainer(const container::KillRequest &);
    static QPair<grpc::Status, container::RestartReply> _restartContainer(const container::RestartRequest &);
    static QPair<grpc::Status, container::UpdateReply> _updateContainer(const container::UpdateRequest &);
    static QPair<grpc::Status, container::RemoveReply> _removeContainer(const container::RemoveRequest &);

    // network management
    static QPair<grpc::Status, network::ListReply> _listNetwork(const network::ListRequest &);

    // image management
    static QPair<grpc::Status, image::ListReply> _listImage(const image::ListRequest &);

signals:
    // node management
    void listNodeFinished(const QPair<grpc::Status, node::ListReply> &);
    void createNodeFinished(const QPair<grpc::Status, node::CreateReply> &);
    void removeNodeFinished(const QPair<grpc::Status, node::RemoveReply> &);
    void nodeStatusFinished(const QPair<grpc::Status, node::StatusReply> &);

    // container management
    void listContainerFinished(const QPair<grpc::Status, container::ListReply> &);
    void createContainerFinished(const QPair<grpc::Status, container::CreateReply> &);
    void containerStatusFinished(const QPair<grpc::Status, container::StatusReply> &);
    void containerInspectFinished(const QPair<grpc::Status, container::InspectReply> &);
    void startContainerFinished(const QPair<grpc::Status, container::StartReply> &);
    void stopContainerFinished(const QPair<grpc::Status, container::StopReply> &);
    void killContainerFinished(const QPair<grpc::Status, container::KillReply> &);
    void restartContainerFinished(const QPair<grpc::Status, container::RestartReply> &);
    void updateContainerFinished(const QPair<grpc::Status, container::UpdateReply> &);
    void removeContainerFinished(const QPair<grpc::Status, container::RemoveReply> &);

    // network management
    void listNetworkFinished(const QPair<grpc::Status, network::ListReply> &);

    // image management
    void listImageFinished(const QPair<grpc::Status, image::ListReply> &);

private:
    QMutex mutex;
};

#endif  // INFOWORKER_H
