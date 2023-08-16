#include "node-list.h"
#include <kiran-log/qt5-log-i.h>
#include "node-addition.h"
#include "rpc.h"

NodeList::NodeList(QWidget *parent) : CommonPage(parent),
                                      m_nodeAddition(nullptr)
{
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
    getNodeList();
}

void NodeList::onCreateNode()
{
    if (!m_nodeAddition)
    {
        m_nodeAddition = new NodeAddition();
        m_nodeAddition->show();
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
}

void NodeList::onMonitor(int row)
{
    KLOG_INFO() << row;
}

void NodeList::getListResult(QPair<grpc::Status, node::ListReply> reply)
{
    KLOG_INFO() << reply.second.nodes_size();
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

    setTableActions(1, QStringList() << ":/images/monitor.svg");

    setTableRowNum(2);
    QStandardItem *item = new QStandardItem("a");
    QStandardItem *itemB = new QStandardItem("b");
    setTableItem(0, 0, item);
    setTableItem(0, 1, itemB);
    setSortableCol(QList<int>() << 0);

    connect(this, &NodeList::sigMonitor, this, &NodeList::onMonitor);
}

void NodeList::getNodeList()
{
    InfoWorker::getInstance().listNode();
    connect(&InfoWorker::getInstance(), &InfoWorker::listNodeFinished, this, &NodeList::getListResult);
}
