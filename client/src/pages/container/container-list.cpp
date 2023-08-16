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
    QList<QMap<QString, QVariant>> info = getCheckedItemInfo(0);
    foreach (auto idMap, info)
    {
        KLOG_INFO() << idMap.value(NODE_ID).toInt();
        KLOG_INFO() << idMap.value(CONTAINER_ID).toString();
    }
}

void ContainerList::onBtnStop()
{
    KLOG_INFO() << "onBtnStop";
}

void ContainerList::onBtnRestart()
{
    KLOG_INFO() << "onBtnRestart";
}

void ContainerList::onBtnDelete()
{
    KLOG_INFO() << "onBtnDelete";
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
            connect(&InfoWorker::getInstance(), &InfoWorker::listContainerFinished, this, &ContainerList::getContainerListResult);
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

                setTableItems(row, 2, QList<QStandardItem *>() << itemStatus << itemImage << itemNodeAddress);

                //                if (i.info().has_resource_stat())
                //                {
                //                    if (i.info().resource_stat().has_cpu_stat())
                //                    {
                //                        double cpuStat = i.info().resource_stat().cpu_stat().core_used().data();
                //                        QString str = QString::number(cpuStat * 100, 'f', 2);
                //                        QStandardItem *itemCpu = new QStandardItem(str);
                //                        itemCpu->setTextAlignment(Qt::AlignCenter);
                //                        setTableItem(5, row, itemCpu);
                //                    }
                //                }

                row++;
            }
        }
    }
    else
    {
        setTableDefaultContent();
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

void ContainerList::insertContainerInfo()
{
}

void ContainerList::getContainerList()
{
    InfoWorker::getInstance().listNode();
    connect(&InfoWorker::getInstance(), &InfoWorker::listNodeFinished, this, &ContainerList::getNodeListResult);
}

void ContainerList::updateInfo(QString keyword)
{
    KLOG_INFO() << "containerList updateInfo";
    //gRPC->拿数据->填充内容
    getContainerList();
    if (!keyword.isEmpty())
        KLOG_INFO() << keyword;
}
