#include "info-worker.h"
#include "rpc.h"

#include <kiran-log/qt5-log-i.h>

#include <QMutexLocker>
#include <QtConcurrent/QtConcurrent>

#define RPC_ASYNC(REPLY_TYPE, WORKER, CALLBACK, ...)                    \
    typedef QPair<grpc::Status, REPLY_TYPE> T;                          \
    QFutureWatcher<T> *watcher = new QFutureWatcher<T>();               \
    watcher->setFuture(QtConcurrent::run(WORKER, ##__VA_ARGS__));       \
    connect(watcher, &QFutureWatcher<T>::finished, [this, watcher] {    \
        auto reply = watcher->result();                                 \
        emit CALLBACK(reply);                                           \
        delete watcher;                                                 \
    });

#define RPC_IMPL(REPLY_TYPE, STUB, RPC_NAME)                            \
    QPair<grpc::Status, REPLY_TYPE> r;                                  \
    auto chan = get_rpc_channel(g_server_addr);                         \
    if (!chan) {                                                        \
        KLOG_INFO("%s %s failed to get connection", #STUB, #RPC_NAME);  \
        r.first = grpc::Status(grpc::StatusCode::UNKNOWN,               \
                        QObject::tr("Network Error").toStdString());    \
        return r;                                                       \
    }                                                                   \
    grpc::ClientContext ctx;                                            \
    r.first = STUB(chan)->RPC_NAME(&ctx, req, &r.second);               \
    return r;



InfoWorker::InfoWorker(QObject *parent) : QObject(parent)
{
    QMutexLocker locker(&mutex);
}

void InfoWorker::listNode()
{
    node::ListRequest req;
    RPC_ASYNC(node::ListReply, _listNode, listNodeFinished, req);
}

void InfoWorker::listContainer(const std::vector<int64_t> &node_ids, const bool all)
{
    container::ListRequest req;
    req.set_list_all(all);
    for (auto &id : node_ids) {
        req.add_node_ids(id);
    }

    RPC_ASYNC(container::ListReply, _listContainer, listContainerFinished, req);
}

void InfoWorker::createNode(const node::CreateRequest &req)
{
    RPC_ASYNC(node::CreateReply, _createNode, createNodeFinished, req);
}

void InfoWorker::createContainer(const container::CreateRequest &req)
{
    RPC_ASYNC(container::CreateReply, _createContainer, createContainerFinished, req);
}

void InfoWorker::containerStatus(const int64_t node_id)
{
    container::StatusRequest req;
    req.set_node_id(node_id);
    RPC_ASYNC(container::StatusReply, _containerStatus, containerStatusFinished, req);
}

void InfoWorker::removeNode(const std::vector<int64_t> &node_ids)
{
    node::RemoveRequest req;
    for (auto &id : node_ids) {
        req.add_ids(id);
    }

    RPC_ASYNC(node::RemoveReply, _removeNode, removeNodeFinished, req);
}

void InfoWorker::nodeStatus(const std::vector<int64_t> &node_ids)
{
    node::StatusRequest req;
    for (auto &id : node_ids) {
        req.add_node_ids(id);
    }

    RPC_ASYNC(node::StatusReply, _nodeStatus, nodeStatusFinished, req);
}

void InfoWorker::containerInspect(const int64_t node_id, const std::string &container_id)
{
    container::InspectRequest req;
    req.set_node_id(node_id);
    req.set_container_id(container_id);
    RPC_ASYNC(container::InspectReply, _containerInspect, containerInspectFinished, req);
}

void InfoWorker::startContainer(const std::map<int64_t, std::vector<std::string>> &ids)
{
    container::StartRequest req;
    for (auto &id : ids)
    {
        auto pId = req.add_ids();
        pId->set_node_id(id.first);
        for (auto &container_id : id.second)
        {
            pId->add_container_ids(container_id);
        }
    }

    RPC_ASYNC(container::StartReply, _startContainer, startContainerFinished, req);
}

void InfoWorker::stopContainer(const std::map<int64_t, std::vector<std::string> > &ids)
{
    container::StopRequest req;
    for (auto &id : ids)
    {
        auto pId = req.add_ids();
        pId->set_node_id(id.first);
        for (auto &container_id : id.second)
        {
            pId->add_container_ids(container_id);
        }
    }

    RPC_ASYNC(container::StopReply, _stopContainer, stopContainerFinished, req);
}

void InfoWorker::killContainer(const std::map<int64_t, std::vector<std::string> > &ids)
{
    container::KillRequest req;
    for (auto &id : ids)
    {
        auto pId = req.add_ids();
        pId->set_node_id(id.first);
        for (auto &container_id : id.second)
        {
            pId->add_container_ids(container_id);
        }
    }

    RPC_ASYNC(container::KillReply, _killContainer, killContainerFinished, req);
}

void InfoWorker::restartContainer(const std::map<int64_t, std::vector<std::string> > &ids)
{
    container::RestartRequest req;
    for (auto &id : ids)
    {
        auto pId = req.add_ids();
        pId->set_node_id(id.first);
        for (auto &container_id : id.second)
        {
            pId->add_container_ids(container_id);
        }
    }

    RPC_ASYNC(container::RestartReply, _restartContainer, restartContainerFinished, req);
}

void InfoWorker::updateContainer(const container::UpdateRequest &req)
{
    RPC_ASYNC(container::UpdateReply, _updateContainer, updateContainerFinished, req);
}

void InfoWorker::removeContainer(const std::map<int64_t, std::vector<std::string> > &ids)
{
    container::RemoveRequest req;
    for (auto &id : ids)
    {
        auto pId = req.add_ids();
        pId->set_node_id(id.first);
        for (auto &container_id : id.second)
        {
            pId->add_container_ids(container_id);
        }
    }
    RPC_ASYNC(container::RemoveReply, _removeContainer, removeContainerFinished, req);
}

void InfoWorker::listNetwork(const int64_t node_id)
{
    network::ListRequest req;
    req.set_node_id(node_id);
    RPC_ASYNC(network::ListReply, _listNetwork, listNetworkFinished, req);
}

void InfoWorker::listImage(const int64_t node_id)
{
    image::ListRequest req;
    req.set_node_id(node_id);
    RPC_ASYNC(image::ListReply, _listImage, listImageFinished, req);
}

QPair<grpc::Status, node::ListReply> InfoWorker::_listNode(const node::ListRequest &req)
{
    RPC_IMPL(node::ListReply, node::Node::NewStub, List);
}

QPair<grpc::Status, container::ListReply> InfoWorker::_listContainer(const container::ListRequest &req)
{
    RPC_IMPL(container::ListReply, container::Container::NewStub, List);
}

QPair<grpc::Status, node::CreateReply> InfoWorker::_createNode(const node::CreateRequest &req)
{
    RPC_IMPL(node::CreateReply, node::Node::NewStub, Create);
}

QPair<grpc::Status, container::CreateReply> InfoWorker::_createContainer(const container::CreateRequest &req)
{
    RPC_IMPL(container::CreateReply, container::Container::NewStub, Create);
}

QPair<grpc::Status, container::StatusReply> InfoWorker::_containerStatus(const container::StatusRequest &req)
{
    RPC_IMPL(container::StatusReply, container::Container::NewStub, Status);
}

QPair<grpc::Status, node::RemoveReply> InfoWorker::_removeNode(const node::RemoveRequest &req)
{
    RPC_IMPL(node::RemoveReply, node::Node::NewStub, Remove);
}

QPair<grpc::Status, node::StatusReply> InfoWorker::_nodeStatus(const node::StatusRequest &req)
{
    RPC_IMPL(node::StatusReply, node::Node::NewStub, Status);
}

QPair<grpc::Status, container::InspectReply> InfoWorker::_containerInspect(const container::InspectRequest &req)
{
    RPC_IMPL(container::InspectReply, container::Container::NewStub, Inspect);
}

QPair<grpc::Status, container::StartReply> InfoWorker::_startContainer(const container::StartRequest &req)
{
    RPC_IMPL(container::StartReply, container::Container::NewStub, Start);
}

QPair<grpc::Status, container::StopReply> InfoWorker::_stopContainer(const container::StopRequest &req)
{
    RPC_IMPL(container::StopReply, container::Container::NewStub, Stop);
}

QPair<grpc::Status, container::KillReply> InfoWorker::_killContainer(const container::KillRequest &req)
{
    RPC_IMPL(container::KillReply, container::Container::NewStub, Kill);
}

QPair<grpc::Status, container::RestartReply> InfoWorker::_restartContainer(const container::RestartRequest &req)
{
    RPC_IMPL(container::RestartReply, container::Container::NewStub, Restart);
}

QPair<grpc::Status, container::UpdateReply> InfoWorker::_updateContainer(const container::UpdateRequest &req)
{
    RPC_IMPL(container::UpdateReply, container::Container::NewStub, Update);
}

QPair<grpc::Status, container::RemoveReply> InfoWorker::_removeContainer(const container::RemoveRequest &req)
{
    RPC_IMPL(container::RemoveReply, container::Container::NewStub, Remove);
}

QPair<grpc::Status, network::ListReply> InfoWorker::_listNetwork(const network::ListRequest &req)
{
    RPC_IMPL(network::ListReply, network::Network::NewStub, List);
}

QPair<grpc::Status, image::ListReply> InfoWorker::_listImage(const image::ListRequest &req)
{
    RPC_IMPL(image::ListReply, image::Image::NewStub, List);
}
