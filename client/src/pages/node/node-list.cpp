#include "node-list.h"
#include <kiran-log/qt5-log-i.h>
#include "common/message-dialog.h"
#include "node-addition.h"
#include "rpc.h"
#define NODE_ID "node id"
#define ACTION_COL 1
NodeList::NodeList(QWidget *parent) : CommonPage(parent),
                                      m_nodeAddition(nullptr)
{
    m_mapStatus.insert(std::pair<int64_t, std::string>(0, tr("Offline").toStdString()));
    m_mapStatus.insert(std::pair<int64_t, std::string>(1, tr("Unknown").toStdString()));
    m_mapStatus.insert(std::pair<int64_t, std::string>(10, tr("Online").toStdString()));
    initButtons();
    initTable();
}

NodeList::~NodeList()
{
    if (m_nodeAddition)
    {
        delete m_nodeAddition;
        m_nodeAddition = nullptr;
    }
}

void NodeList::updateInfo(QString keyword)
{
    KLOG_INFO() << "NodeList updateInfo";
    clearText();
    InfoWorker::getInstance().disconnect();
    if (keyword.isEmpty())
    {
        initNodeConnect();
        getNodeList();
    }
}

void NodeList::onCreateNode()
{
    if (!m_nodeAddition)
    {
        m_nodeAddition = new NodeAddition();
        m_nodeAddition->show();
        connect(m_nodeAddition, &NodeAddition::sigSave, this, &NodeList::onSaveSlot);
        connect(m_nodeAddition, &NodeAddition::destroyed,
                [=] {
                    KLOG_INFO() << " m_nodeAdditiong destroy";
                    m_nodeAddition->deleteLater();
                    m_nodeAddition = nullptr;
                });
    }
}

void NodeList::onRemoveNode()
{
    KLOG_INFO() << "onRemoveNode";
    QList<QMap<QString, QVariant>> info = getCheckedItemInfo(0);
    std::vector<int64_t> node_ids;
    foreach (auto &idMap, info)
    {
        KLOG_INFO() << idMap.value(NODE_ID).toInt();
        node_ids.push_back(idMap.value(NODE_ID).toInt());
    }

    if (!node_ids.empty())
    {
        MessageDialog::StandardButton ret = MessageDialog::message(tr("Delete Node"),
                                                                   tr("Are you sure you want to delete the node?"),
                                                                   tr("It can't be recovered after deletion.Are you sure you want to continue?"),
                                                                   ":/images/warning.png",
                                                                   MessageDialog::StandardButton::Yes | MessageDialog::StandardButton::Cancel);
        if (ret == MessageDialog::StandardButton::Yes)
        {
            InfoWorker::getInstance().removeNode(node_ids);
        }
        else
            KLOG_INFO() << "cancel";
    }
}

void NodeList::onMonitor(int row)
{
    KLOG_INFO() << row;
}

void NodeList::onSaveSlot(QMap<QString, QString> Info)
{
    KLOG_INFO() << "name" << Info["Node Name"] << "ip" << Info["Node IP"];
    node::CreateRequest request;
    request.set_name(Info["Node Name"].toStdString());
    request.set_address(Info["Node IP"].toStdString());
    InfoWorker::getInstance().createNode(request);
}

void NodeList::getListResult(const QPair<grpc::Status, node::ListReply> &reply)
{
    KLOG_INFO() << reply.second.nodes_size();
    if (reply.first.ok())
    {
        int size = reply.second.nodes_size();
        if (size <= 0)
        {
            setTableDefaultContent(QList<int>() << ACTION_COL, "-");
            setOpBtnEnabled(false);
            return;
        }

        clearTable();
        setOpBtnEnabled(true);
        int row = 0;
        QMap<QString, QVariant> idMap;
        for (auto node : reply.second.nodes())
        {
            KLOG_INFO() << "nodeid:" << node.id();
            qint64 nodeId = node.id();
            idMap.insert(NODE_ID, nodeId);

            QStandardItem *itemName = new QStandardItem(node.name().data());
            itemName->setData(QVariant::fromValue(idMap));
            itemName->setCheckable(true);

            QStandardItem *itemIp = new QStandardItem(node.address().data());
            itemIp->setTextAlignment(Qt::AlignCenter);

            std::string state = m_mapStatus[1];
            std::string strCntrCnt = "-/-";
            std::string strCpuPct = "-";
            std::string strMemPct = "-";
            if (node.has_status())
            {
                state = m_mapStatus[node.status().state()];
                auto &status = node.status();
                if (status.has_container_stat())
                    strCntrCnt = std::to_string(status.container_stat().running()) + "/" + std::to_string(status.container_stat().total());

                if (status.has_cpu_stat())
                {
                    char str[128]{};
                    sprintf(str, "%0.1f%%", status.cpu_stat().used() * 100);
                    strCpuPct = std::string(str);
                }

                if (status.has_mem_stat())
                {
                    char str[128]{};
                    sprintf(str, "%0.1f%%", status.mem_stat().used_percentage());
                    strMemPct = std::string(str);
                }
            }

            QStandardItem *itemStatus = new QStandardItem(state.data());
            itemStatus->setTextAlignment(Qt::AlignCenter);
            QStandardItem *itemCntrCnt = new QStandardItem(strCntrCnt.data());
            itemCntrCnt->setTextAlignment(Qt::AlignCenter);
            QStandardItem *itemCpu = new QStandardItem(strCpuPct.data());
            itemCpu->setTextAlignment(Qt::AlignCenter);
            QStandardItem *itemMem = new QStandardItem(strMemPct.data());
            itemMem->setTextAlignment(Qt::AlignCenter);
            QStandardItem *itemDisk = new QStandardItem("-");
            itemDisk->setTextAlignment(Qt::AlignCenter);

            setTableItem(row, 0, itemName);
            setTableItems(row, 2, QList<QStandardItem *>() << itemStatus << itemIp << itemCntrCnt << itemCpu << itemMem << itemDisk);
            row++;
        }
    }
    else
    {
        setTableDefaultContent(QList<int>() << ACTION_COL, "-");
        setOpBtnEnabled(false);
    }
}

void NodeList::getCreateResult(const QPair<grpc::Status, node::CreateReply> &reply)
{
    KLOG_INFO() << reply.first.error_code() << reply.first.error_message().data();
    if (reply.first.ok())
    {
        getNodeList();
        return;
    }
}

void NodeList::getRemoveResult(const QPair<grpc::Status, node::RemoveReply> &reply)
{
    KLOG_INFO() << reply.first.error_code() << reply.first.error_message().data();
    if (reply.first.ok())
    {
        getNodeList();
    }
}

void NodeList::initButtons()
{
    QPushButton *btnCreate = new QPushButton(this);
    btnCreate->setText(tr("Create"));
    btnCreate->setObjectName("btnCreate");
    btnCreate->setFixedSize(QSize(100, 40));
    connect(btnCreate, &QPushButton::clicked, this, &NodeList::onCreateNode);

    QPushButton *btnRemove = new QPushButton(this);
    btnRemove->setText(tr("Remove"));
    btnRemove->setObjectName("btnRemove");
    btnRemove->setFixedSize(QSize(100, 40));
    connect(btnRemove, &QPushButton::clicked, this, &NodeList::onRemoveNode);

    addOperationButtons(QList<QPushButton *>() << btnCreate << btnRemove);
    setOpBtnEnabled(false);
}

void NodeList::initTable()
{
    QStringList tableHHeaderDate = {QString(tr("Node Name")),
                                    QString(tr("Quick Actions")),
                                    QString(tr("Status")),
                                    QString(tr("IP")),
                                    QString(tr("Container Number")),
                                    "CPU",
                                    QString(tr("Memory")),
                                    QString(tr("Disk"))};
    setHeaderSections(tableHHeaderDate);
    setTableColNum(tableHHeaderDate.size());
    QList<int> sortablCol = {0, 2};
    setSortableCol(sortablCol);
    setTableActions(ACTION_COL, QStringList() << ":/images/monitor.svg");
    setTableDefaultContent(QList<int>() << ACTION_COL, "-");

    connect(this, &NodeList::sigMonitor, this, &NodeList::onMonitor);
}

void NodeList::initNodeConnect()
{
    connect(&InfoWorker::getInstance(), &InfoWorker::listNodeFinished, this, &NodeList::getListResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::createNodeFinished, this, &NodeList::getCreateResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::removeNodeFinished, this, &NodeList::getRemoveResult);
}

void NodeList::getNodeList()
{
    InfoWorker::getInstance().listNode();
}
