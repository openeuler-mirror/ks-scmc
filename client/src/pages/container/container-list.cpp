#include "container-list.h"
#include <kiran-log/qt5-log-i.h>
#include <QComboBox>
#include <QHBoxLayout>
#include <QLabel>
#include <QMouseEvent>
#include <QStandardItem>
#include "container-setting.h"

#define NODE_ID "node id"
#define CONTAINER_ID "container id"

ContainerList::ContainerList(QWidget *parent)
    : CommonPage(parent),
      m_createCTSetting(nullptr),
      m_editCTSetting(nullptr)
{
    initButtons();
    //初始化表格
    initTable();
    initConnect();
}

ContainerList::~ContainerList()
{
    if (m_createCTSetting)
    {
        delete m_createCTSetting;
        m_createCTSetting = nullptr;
    }
    if (m_editCTSetting)
    {
        delete m_editCTSetting;
        m_editCTSetting = nullptr;
    }
}

void ContainerList::onBtnCreate()
{
    KLOG_INFO() << "onBtnCreate";
    if (!m_createCTSetting)
    {
        m_createCTSetting = new ContainerSetting(CONTAINER_SETTING_TYPE_CONTAINER_CREATE);
        //initContianerSetting(m_createCTSetting, CONTAINER_SETTING_TYPE_CREATE);
        m_createCTSetting->show();
        connect(m_createCTSetting, &ContainerSetting::destroyed,
                [=] {
                    KLOG_INFO() << "create container setting destroy";
                    m_createCTSetting->deleteLater();
                    m_createCTSetting = nullptr;
                });
    }
}

void ContainerList::onBtnRun()
{
    KLOG_INFO() << "onBtnRun";
    std::map<int64_t, std::vector<std::string>> ids;
    getCheckId(ids);
    InfoWorker::getInstance().startContainer(ids);
}

void ContainerList::onBtnStop()
{
    KLOG_INFO() << "onBtnStop";

    std::map<int64_t, std::vector<std::string>> ids;
    getCheckId(ids);
    InfoWorker::getInstance().stopContainer(ids);
}

void ContainerList::onBtnRestart()
{
    KLOG_INFO() << "onBtnRestart";
    std::map<int64_t, std::vector<std::string>> ids;
    getCheckId(ids);
    InfoWorker::getInstance().restartContainer(ids);
}

void ContainerList::onBtnDelete()
{
    KLOG_INFO() << "onBtnDelete";
    std::map<int64_t, std::vector<std::string>> ids;
    getCheckId(ids);
    InfoWorker::getInstance().removeContainer(ids);
}

void ContainerList::onActCopyConfig()
{
    KLOG_INFO() << "onCopyConfig";
}

void ContainerList::onActBatchUpdate()
{
    KLOG_INFO() << "onBatchUpdate";
}

void ContainerList::onActBatchEdit()
{
    KLOG_INFO() << "onBatchEdit";
}

void ContainerList::onActBackup()
{
    KLOG_INFO() << "onBackup";
}

void ContainerList::onMonitor(int row)
{
    KLOG_INFO() << "ContainerList::onMonitor" << row;
}

void ContainerList::onEdit(int row)
{
    KLOG_INFO() << row;
    if (!m_editCTSetting)
    {
        m_editCTSetting = new ContainerSetting(CONTAINER_SETTING_TYPE_CONTAINER_EDIT);
        m_editCTSetting->show();
        connect(m_editCTSetting, &ContainerSetting::destroyed,
                [=] {
                    KLOG_INFO() << " edit container setting destroy";
                    m_editCTSetting->deleteLater();
                    m_editCTSetting = nullptr;
                });
    }
}

void ContainerList::onTerminal(int row)
{
    KLOG_INFO() << row;
}

void ContainerList::getNodeListResult(QPair<grpc::Status, node::ListReply> reply)
{
    KLOG_INFO() << "getNodeListResult";
    if (reply.first.ok())
    {
        m_vecNodeId.clear();
        for (auto n : reply.second.nodes())
        {
            KLOG_INFO() << n.id();
            m_vecNodeId.push_back(n.id());
        }
        if (!m_vecNodeId.empty())
        {
            InfoWorker::getInstance().listContainer(m_vecNodeId, true);
        }
    }
    else
        setTableDefaultContent();
}

void ContainerList::getContainerListResult(QPair<grpc::Status, container::ListReply> reply)
{
    KLOG_INFO() << "getContainerListResult";
    if (reply.first.ok())
    {
        clearTable();
        int size = reply.second.containers_size();
        KLOG_INFO() << "container size:" << size;
        if (size > 0)
        {
            int row = 0;
            QMap<QString, QVariant> idMap;
            for (auto i : reply.second.containers())
            {
                qint64 nodeId = i.node_id();
                idMap.insert(NODE_ID, nodeId);
                idMap.insert(CONTAINER_ID, i.info().id().data());

                QStandardItem *itemName = new QStandardItem(i.info().name().data());
                itemName->setData(QVariant::fromValue(idMap));
                itemName->setCheckable(true);
                setTableItem(row, 0, itemName);

                QStandardItem *itemStatus = new QStandardItem(i.info().state().data());
                itemStatus->setTextAlignment(Qt::AlignCenter);

                QStandardItem *itemImage = new QStandardItem(i.info().image().data());
                itemImage->setTextAlignment(Qt::AlignCenter);

                QStandardItem *itemNodeAddress = new QStandardItem(i.node_address().data());
                itemNodeAddress->setTextAlignment(Qt::AlignCenter);



                std::string strCpuPct = "-";
                std::string strMemPct = "-";

                if (i.info().has_resource_stat())
                {
                    if (i.info().resource_stat().has_cpu_stat())
                    {
                        char str[128]{};
                        sprintf(str, "%0.1f%%", i.info().resource_stat().cpu_stat().core_used()*100);
                        strCpuPct = std::string(str);
                    }

                    if (i.info().resource_stat().has_mem_stat())
                    {
                        double used = i.info().resource_stat().mem_stat().used()/1048576;
                        char str[128]{};
                        sprintf(str, "%0.0fMB", used);
                        strMemPct = std::string(str);
                    }
                }

                QStandardItem *itemCpu= new QStandardItem(strCpuPct.data());
                itemCpu->setTextAlignment(Qt::AlignCenter);
                QStandardItem *itemMem = new QStandardItem(strMemPct.data());
                QStandardItem *itemDisk = new QStandardItem("-");
                itemDisk->setTextAlignment(Qt::AlignCenter);
                itemMem->setTextAlignment(Qt::AlignCenter);
                setTableItems(row, 2, QList<QStandardItem *>() << itemStatus << itemImage << itemNodeAddress
                                                               << itemCpu << itemMem << itemDisk);

                row++;
            }
        }
    }
    else
    {
        setTableDefaultContent();
    }
}

void ContainerList::getContainerStartResult(QPair<grpc::Status, container::StartReply> reply)
{
    KLOG_INFO() << reply.first.error_code() << reply.first.error_message().data();
    if (reply.first.ok())
    {
        getContainerList();
        return ;
    }
}

void ContainerList::getContainerStopResult(QPair<grpc::Status, container::StopReply> reply)
{
    KLOG_INFO() << reply.first.error_code() << reply.first.error_message().data();
    if (reply.first.ok())
    {
        getContainerList();
        return ;
    }
}

void ContainerList::getContainerRestartResult(QPair<grpc::Status, container::RestartReply> reply)
{
    KLOG_INFO() << reply.first.error_code() << reply.first.error_message().data();
    if (reply.first.ok())
    {
        getContainerList();
        return ;
    }
}

void ContainerList::getContainerRemoveResult(QPair<grpc::Status, container::RemoveReply> reply)
{
    KLOG_INFO() << reply.first.error_code() << reply.first.error_message().data();
    if (reply.first.ok())
    {
        getContainerList();
        return ;
    }
}

void ContainerList::initButtons()
{
    //创建按钮及菜单
    QToolButton *btnCreate = new QToolButton(this);
    btnCreate->setText(tr("Create"));
    btnCreate->setObjectName("btnCreate");
    btnCreate->setFixedSize(QSize(100, 40));
    btnCreate->setPopupMode(QToolButton::MenuButtonPopup);
    connect(btnCreate, &QToolButton::clicked, this, &ContainerList::onBtnCreate);
    addOperationButton(btnCreate);

    QMenu *btnCreateMenu = new QMenu(btnCreate);
    btnCreateMenu->setObjectName("btnCreateMenu");
    QAction *copyConf = btnCreateMenu->addAction(tr("Copy configuration"));
    connect(copyConf, &QAction::triggered, this, &ContainerList::onActCopyConfig);
    btnCreate->setMenu(btnCreateMenu);

    //其他按钮及菜单
    const QMap<int, QString> btnNameMap = {
        {OPERATION_BUTTOM_RUN, tr("Run")},
        {OPERATION_BUTTOM_STOP, tr("Stop")},
        {OPERATION_BUTTOM_RESTART, tr("Restart")},
        {OPERATION_BUTTOM_DELETE, tr("Delete")},
        {OPERATION_BUTTOM_MORE, tr("More")}};

    for (auto iter = btnNameMap.begin(); iter != btnNameMap.end(); iter++)
    {
        QString name = iter.value();
        QPushButton *btn = new QPushButton(this);
        btn->setObjectName(QString("btn%1").arg(name));
        btn->setText(name);
        btn->setFixedSize(QSize(100, 40));
        m_opBtnMap.insert(iter.key(), btn);
    }

    QPushButton *btnMore = m_opBtnMap[OPERATION_BUTTOM_MORE];
    QMenu *moreMenu = new QMenu(this);
    moreMenu->setObjectName("moreMenu");
    QAction *actBatchUpdate = moreMenu->addAction(tr("Batch update version"));
    QAction *actBatchEdit = moreMenu->addAction(tr("Batch edit"));
    QAction *actBackup = moreMenu->addAction(tr("Backupt"));
    btnMore->setMenu(moreMenu);

    connect(actBatchUpdate, &QAction::triggered, this, &ContainerList::onActBatchUpdate);
    connect(actBatchEdit, &QAction::triggered, this, &ContainerList::onActBatchEdit);
    connect(actBackup, &QAction::triggered, this, &ContainerList::onActBackup);

    connect(m_opBtnMap[OPERATION_BUTTOM_RUN], &QPushButton::clicked, this, &ContainerList::onBtnRun);
    connect(m_opBtnMap[OPERATION_BUTTOM_STOP], &QPushButton::clicked, this, &ContainerList::onBtnStop);
    connect(m_opBtnMap[OPERATION_BUTTOM_RESTART], &QPushButton::clicked, this, &ContainerList::onBtnRestart);
    connect(m_opBtnMap[OPERATION_BUTTOM_DELETE], &QPushButton::clicked, this, &ContainerList::onBtnDelete);

    addOperationButtons(m_opBtnMap.values());
}

void ContainerList::initTable()
{
    QStringList tableHHeaderDate = {QString(tr("Container Name")),
                                    QString(tr("Quick Actions")),
                                    QString(tr("Status")),
                                    QString(tr("Image")),
                                    QString(tr("Node")),
                                    "CPU",
                                    QString(tr("Memory")),
                                    QString(tr("Disk")),
                                    QString(tr("Online Time"))};
    setHeaderSections(tableHHeaderDate);
    QList<int> sortablCol = {0, 3};
    setSortableCol(sortablCol);
    setTableActions(1, QStringList() << ":/images/monitor.svg"
                                     << ":/images/edit.svg"
                                     << ":/images/terminal.svg"
                                     << ":/images/more_in_table.svg");

    connect(this, &ContainerList::sigMonitor, this, &ContainerList::onMonitor);
    connect(this, &ContainerList::sigEdit, this, &ContainerList::onEdit);
}

void ContainerList::initConnect()
{
    connect(&InfoWorker::getInstance(), &InfoWorker::listNodeFinished, this, &ContainerList::getNodeListResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::listContainerFinished, this, &ContainerList::getContainerListResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::startContainerFinished, this, &ContainerList::getContainerStartResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::stopContainerFinished, this, &ContainerList::getContainerStopResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::restartContainerFinished, this, &ContainerList::getContainerRestartResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::removeContainerFinished, this, &ContainerList::getContainerRemoveResult);
}

void ContainerList::insertContainerInfo()
{
}

void ContainerList::getContainerList()
{
    InfoWorker::getInstance().listNode();
}

void ContainerList::getCheckId(std::map<int64_t, std::vector<std::string>> &ids)
{
    QList<QMap<QString, QVariant>> info = getCheckedItemInfo(0);
    int64_t node_id{};

    foreach (auto idMap, info)
    {
        KLOG_INFO() << idMap.value(NODE_ID).toInt();
        KLOG_INFO() << idMap.value(CONTAINER_ID).toString();

        node_id = idMap.value(NODE_ID).toInt();
        std::map<int64_t, std::vector<std::string>>::iterator iter = ids.find(node_id);
        if (iter == ids.end())
        {
            std::vector<std::string> container_ids;
            container_ids.push_back(idMap.value(CONTAINER_ID).toString().toStdString());
            ids.insert(std::pair<int64_t, std::vector<std::string>>(node_id, container_ids));
        }
        else
        {
            ids[node_id].push_back(idMap.value(CONTAINER_ID).toString().toStdString());
        }
    }

}

void ContainerList::updateInfo(QString keyword)
{
    KLOG_INFO() << "containerList updateInfo";
    initConnect();
    //gRPC->拿数据->填充内容
    getContainerList();
    if (!keyword.isEmpty())
        KLOG_INFO() << keyword;
}
