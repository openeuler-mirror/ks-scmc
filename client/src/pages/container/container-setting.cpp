#include "container-setting.h"
#include <kiran-log/qt5-log-i.h>
#include <QComboBox>
#include <QMenu>
#include <QMouseEvent>
#include <QPainter>
#include <QPair>
#include "advanced-configuration/envs-conf-page.h"
#include "advanced-configuration/graphic-conf-page.h"
#include "advanced-configuration/high-availability-page.h"
#include "advanced-configuration/volumes-conf-page.h"
#include "base-configuration/cpu-conf-page.h"
#include "base-configuration/memory-conf-page.h"
#include "base-configuration/network-conf-page.h"
#include "common/def.h"
#include "common/guide-item.h"
#include "common/message-dialog.h"
#include "ui_container-setting.h"

#define CPU "CPU"
#define MEMORY "Memory"
#define NETWORK_CARD "Network Card"
#define ENVS "ENVS"
#define GRAPHIC "Graphic"
#define VOLUMES "Volumes"
#define HIGH_AVAILABILITY "High availability"

#define NODE_ID "node id"
#define NODE_ADDRESS "node address"

const std::string TagContainerDescription = "TAG_CONTAINER_DESC";

ContainerSetting::ContainerSetting(ContainerSettingType type, QWidget *parent) : QWidget(parent),
                                                                                 ui(new Ui::ContainerSetting),
                                                                                 m_netWorkCount(0),
                                                                                 m_type(type)
{
    ui->setupUi(this);
    initUI();
    setAttribute(Qt::WA_DeleteOnClose);

    connect(&InfoWorker::getInstance(), &InfoWorker::listNodeFinished, this, &ContainerSetting::getNodeListResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::createContainerFinished, this, &ContainerSetting::getCreateContainerResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::containerInspectFinished, this, &ContainerSetting::getContainerInspectResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::updateContainerFinished, this, &ContainerSetting::getUpdateContainerResult);
    connect(&InfoWorker::getInstance(), &InfoWorker::listImageFinished, this, &ContainerSetting::getListImageFinishedResult);

    getNodeInfo();
}

ContainerSetting::~ContainerSetting()
{
    delete ui;
}

void ContainerSetting::paintEvent(QPaintEvent *event)
{
    Q_UNUSED(event);
    QStyleOption opt;
    opt.init(this);
    QPainter p(this);
    style()->drawPrimitive(QStyle::PE_Widget, &opt, &p, this);
}

void ContainerSetting::setItems(int row, int col, QWidget *item)
{
    ui->gridLayout->addWidget(item, row, col);
}

void ContainerSetting::setContainerNodeIds(QPair<int64_t, QString> ids)
{
    m_containerIds = ids;
}

bool ContainerSetting::eventFilter(QObject *obj, QEvent *ev)
{
    QMouseEvent *mouseEvent = static_cast<QMouseEvent *>(ev);
    if (obj == ui->btn_add && mouseEvent->type() == QEvent::MouseButtonPress)
    {
        int x = ui->btn_add->width() / 2 - m_addMenu->sizeHint().width() / 2;
        int y = -m_addMenu->sizeHint().height() - 2;
        QPoint menuPos(x, y);
        m_addMenu->popup(ui->btn_add->mapToGlobal(menuPos));
        return true;
    }
    return false;
}

void ContainerSetting::initUI()
{
    setWindowIcon(QIcon(":/images/logo.png"));
    setWindowModality(Qt::ApplicationModal);
    ui->tabWidget->setStyleSheet(QString("QTabWidget::tab-bar{width:%1px;}").arg(this->geometry().width() + 20));
    ui->tabWidget->setFocusPolicy(Qt::NoFocus);
    ui->btn_add->setIcon(QIcon(":/images/addition.svg"));
    ui->btn_add->setText(tr("Add"));
    ui->btn_add->installEventFilter(this);
    m_addMenu = new QMenu(this);
    QAction *act = m_addMenu->addAction(tr("Network"));
    act->setData(TAB_CONFIG_GUIDE_ITEM_TYP_NETWORK_CARD);
    m_addMenu->setObjectName("addMenu");
    connect(m_addMenu, &QMenu::triggered, this, &ContainerSetting::onAddItem);

    m_baseConfStack = new QStackedWidget(ui->tab_base_config);
    QLayout *baseLayout = ui->tab_base_config->layout();
    baseLayout->addWidget(m_baseConfStack);

    m_advancedConfStack = new QStackedWidget(ui->tab_advanced_config);
    QLayout *advancedLayout = ui->tab_advanced_config->layout();
    advancedLayout->addWidget(m_advancedConfStack);

    //创建tabwidget中侧边栏
    QList<QPair<QString, QString>> baseConfItemInfo = {{tr("CPU"), ":/images/container-cpu.svg"},
                                                       {tr("Memory"), ":/images/container-memory.svg"},
                                                       {tr("Network card"), ":/images/container-net-card.svg"}};
    for (int i = 0; i < baseConfItemInfo.count(); i++)
    {
        QString name = baseConfItemInfo.at(i).first;
        GuideItem *item = createGuideItem(ui->listwidget_base_config,
                                          name,
                                          GUIDE_ITEM_TYPE_NORMAL,
                                          baseConfItemInfo.at(i).second);
        m_baseItemMap.append(item);
    }

    QList<QPair<QString, QString>> advancedConfItemInfo = {{tr("Envs"), ":/images/container-env.png"},
                                                           {tr("Graphic"), ":/images/audit-center.svg"},
                                                           {tr("Volumes"), ":/images/container-volumes.png"},
                                                           {tr("High availability"), ":/images/container-high-avail.png"}};
    for (int i = 0; i < advancedConfItemInfo.count(); i++)
    {
        QString name = advancedConfItemInfo.at(i).first;
        GuideItem *item = createGuideItem(ui->listWidget_advanced_config,
                                          name,
                                          GUIDE_ITEM_TYPE_NORMAL,
                                          advancedConfItemInfo.at(i).second);
        m_advancedItemMap.append(item);
    }

    initSummaryUI();
    initBaseConfPages();
    initAdvancedConfPages();

    QList<QComboBox *> cbList = this->findChildren<QComboBox *>();
    foreach (auto cb, cbList)
    {
        cb->setItemDelegate(new QStyledItemDelegate(cb));
    }

    if (m_type == CONTAINER_SETTING_TYPE_CONTAINER_EDIT)
    {
        ui->listwidget_base_config->setItemHidden(ui->listwidget_base_config->item(TAB_CONFIG_GUIDE_ITEM_TYP_NETWORK_CARD), true);
        ui->listWidget_advanced_config->setItemHidden(ui->listWidget_advanced_config->item(TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_ENVS), true);
        ui->listWidget_advanced_config->setItemHidden(ui->listWidget_advanced_config->item(TAB_CONFIG_GUIDE_ITEM_TYPDE_ITEM_GRAPHIC), true);
        ui->listWidget_advanced_config->setItemHidden(ui->listWidget_advanced_config->item(TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_VOLUMES), true);
        m_advancedConfStack->setCurrentIndex(TAB_CONFIG_GUIDE_ITEM_TYP_HIGH_AVAILABILITY);
        ui->btn_add->hide();
    }

    connect(ui->listwidget_base_config, &QListWidget::itemClicked, this, &ContainerSetting::onItemClicked);
    connect(ui->listWidget_advanced_config, &QListWidget::itemClicked, this, &ContainerSetting::onItemClicked);
    connect(ui->btn_confirm, &QToolButton::clicked, this, &ContainerSetting::onConfirm);
    connect(ui->btn_cancel, &QToolButton::clicked, this, &ContainerSetting::close);
    connect(ui->cb_node, &QComboBox::currentTextChanged, this, &ContainerSetting::onNodeSelectedChanged);
}

void ContainerSetting::initSummaryUI()
{
    switch (m_type)
    {
    case CONTAINER_SETTING_TYPE_CONTAINER_CREATE:
    {
        setWindowTitle(tr("Create Container"));
        m_cbImage = new QComboBox(this);
        m_cbImage->setFixedSize(QSize(200, 30));
        QGridLayout *layout = dynamic_cast<QGridLayout *>(ui->page_container->layout());
        layout->addWidget(m_cbImage, 2, 1);
        ui->stackedWidget->setCurrentWidget(ui->page_container);
        break;
    }
    case CONTAINER_SETTING_TYPE_CONTAINER_EDIT:
    {
        setWindowTitle(tr("Edit Container"));
        m_labImage = new QLabel(this);
        QGridLayout *layout = dynamic_cast<QGridLayout *>(ui->page_container->layout());
        layout->addWidget(m_labImage, 2, 1);
        ui->lineEdit_name->setReadOnly(true);
        ui->lineEdit_name->setStyleSheet("#lineEdit_name{border:none;background:#ffffff;}");
        ui->stackedWidget->setCurrentWidget(ui->page_container);
        break;
    }
    case CONTAINER_SETTING_TYPE_TEMPLATE_CREATE:
    case CONTAINER_SETTING_TYPE_TEMPLATE_EDIT:
        ui->stackedWidget->setCurrentWidget(ui->page_template);
    default:
        break;
    }
}

GuideItem *ContainerSetting::createGuideItem(QListWidget *parent, QString text, int type, QString icon)
{
    QListWidgetItem *newItem = nullptr;
    GuideItem *customItem = nullptr;

    newItem = new QListWidgetItem(parent);

    customItem = new GuideItem(text, icon, type, parent);
    if (text == NETWORK_CARD)
    {
        customItem->setDeleteBtn();
        m_netWorkCount++;
        if (m_netWorkCount == 1)
            customItem->setDeleteBtnVisible(false);
        connect(customItem, &GuideItem::sigDeleteItem, this, &ContainerSetting::onDelItem);
    }
    customItem->setTipLinePosition(TIP_LINE_POSITION_RIGHT);
    parent->addItem(newItem);
    parent->setItemWidget(newItem, customItem);
    newItem->setTextAlignment(Qt::AlignVCenter);
    newItem->setSizeHint(QSize(220, 30));
    return customItem;
}

void ContainerSetting::initBaseConfPages()
{
    CPUConfPage *cpuInfoPage = new CPUConfPage(m_totalCPU, ui->tab_base_config);
    m_baseConfStack->addWidget(cpuInfoPage);

    MemoryConfPage *memoryConfPage = new MemoryConfPage(ui->tab_base_config);
    m_baseConfStack->addWidget(memoryConfPage);

    NetworkConfPage *networkConfPage = new NetworkConfPage(ui->tab_base_config);
    m_baseConfStack->addWidget(networkConfPage);
}

void ContainerSetting::initAdvancedConfPages()
{
    EnvsConfPage *envsConfPage = new EnvsConfPage(ui->tab_advanced_config);
    m_advancedConfStack->addWidget(envsConfPage);

    GraphicConfPage *graphicConfPage = new GraphicConfPage(ui->tab_advanced_config);
    m_advancedConfStack->addWidget(graphicConfPage);

    VolumesConfPage *volumesConfPage = new VolumesConfPage(ui->tab_advanced_config);
    m_advancedConfStack->addWidget(volumesConfPage);

    HighAvailabilityPage *highAvailability = new HighAvailabilityPage(ui->tab_advanced_config);
    m_advancedConfStack->addWidget(highAvailability);
}

void ContainerSetting::updateRemovableItem(QString itemText)
{
    if (m_netWorkCount > 1)
    {
        foreach (GuideItem *item, m_baseItemMap)
        {
            if (item->getItemText() == itemText)
            {
                item->setDeleteBtnVisible(true);
            }
        }
    }
    else if (m_netWorkCount == 1)
    {
        foreach (GuideItem *item, m_baseItemMap)
        {
            if (item->getItemText() == itemText)
            {
                item->setDeleteBtnVisible(false);
            }
        }
    }
}

void ContainerSetting::getNodeInfo()
{
    InfoWorker::getInstance().listNode();
}

void ContainerSetting::getImageInfo(int64_t node_id)
{
    KLOG_INFO() << "getImageInfo";
    KLOG_INFO() << m_containerIds.first;
    InfoWorker::getInstance().listImage(node_id);
}

void ContainerSetting::createContainer()
{
    container::CreateRequest request;
    ErrorCode ret;
    request.set_node_id(m_nodeInfo.key(ui->cb_node->currentText()));
    request.set_name(ui->lineEdit_name->text().toStdString());
    auto cntrCfg = request.mutable_config();
    if (!ui->lineEdit_describe->text().isEmpty())
    {
        auto labels = cntrCfg->mutable_labels();
        labels->insert({TagContainerDescription, ui->lineEdit_describe->text().toStdString()});
    }
    cntrCfg->set_image(m_cbImage->currentText().toStdString());

    //cpu
    auto hostCfg = request.mutable_host_config();
    auto resourceCfg = hostCfg->mutable_resource_config();
    auto cpuPage = qobject_cast<CPUConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYPE_CPU));
    cpuPage->getCPUInfo(resourceCfg);

    //memory
    auto memoryPage = qobject_cast<MemoryConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_MEMORY));
    ret = memoryPage->getMemoryInfo(resourceCfg);
    if (ret == INPUT_ARG_ERROR)
    {
        MessageDialog::message(tr("Memory Data"),
                               tr("Input error"),
                               tr("Memory soft limit can't be greater than the maximum limit !"),
                               tr(":/images/error.svg"),
                               MessageDialog::StandardButton::Ok);
        return;
    }

    //High
    auto policy = hostCfg->mutable_restart_policy();
    auto highAvailabilityPage = qobject_cast<HighAvailabilityPage *>(m_advancedConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_HIGH_AVAILABILITY));
    highAvailabilityPage->getRestartPolicy(policy);

    //network card
    auto networkPage = qobject_cast<NetworkConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_NETWORK_CARD));
    networkPage->getNetworkInfo(&request);

    //env
    auto envPage = qobject_cast<EnvsConfPage *>(m_advancedConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_ENVS));
    ret = envPage->getEnvInfo(cntrCfg);
    if (ret == INPUT_NULL_ERROR)
    {
        MessageDialog::message(tr("Env Data"),
                               tr("Input error"),
                               tr("Please improve the contents in Env table!"),
                               tr(":/images/error.svg"),
                               MessageDialog::StandardButton::Ok);
        return;
    }

    //volume
    auto volumePage = qobject_cast<VolumesConfPage *>(m_advancedConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_VOLUMES));
    ret = volumePage->getVolumeInfo(hostCfg);
    if (ret == INPUT_NULL_ERROR)
    {
        MessageDialog::message(tr("Volumes Data"),
                               tr("Input error"),
                               tr("Please improve the contents in Volumes table!"),
                               tr(":/images/error.svg"),
                               MessageDialog::StandardButton::Ok);
        return;
    }

    //Graph
    auto graphicPage = qobject_cast<GraphicConfPage *>(m_advancedConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYPDE_ITEM_GRAPHIC));
    graphicPage->getGraphicInfo(&request);

    InfoWorker::getInstance().createContainer(request);
}

void ContainerSetting::updateContainer()
{
    container::UpdateRequest request;
    request.set_node_id(m_containerIds.first);
    request.set_container_id(m_containerIds.second.toStdString());

    auto rsrcCfg = request.mutable_resource_config();
    auto cpuPage = qobject_cast<CPUConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYPE_CPU));
    cpuPage->getCPUInfo(rsrcCfg);

    auto memoryPage = qobject_cast<MemoryConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_MEMORY));
    memoryPage->getMemoryInfo(rsrcCfg);

    auto policy = request.mutable_restart_policy();
    auto highAvailabilityPage = qobject_cast<HighAvailabilityPage *>(m_advancedConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_HIGH_AVAILABILITY));
    highAvailabilityPage->getRestartPolicy(policy);

    InfoWorker::getInstance().updateContainer(request);
}

void ContainerSetting::onItemClicked(QListWidgetItem *item)
{
    QListWidget *listwidget = qobject_cast<QListWidget *>(sender());
    GuideItem *guideItem = qobject_cast<GuideItem *>(listwidget->itemWidget(item));
    int index = listwidget->row(item);
    if (listwidget == ui->listwidget_base_config)
    {
        m_baseConfStack->setCurrentIndex(index);
        foreach (GuideItem *item, m_baseItemMap)
        {
            if (item == guideItem)
                item->setSelected(true);
            else
                item->setSelected(false);
        }
    }

    else if (listwidget == ui->listWidget_advanced_config)
    {
        m_advancedConfStack->setCurrentIndex(index);
        foreach (GuideItem *item, m_advancedItemMap)
        {
            if (item == guideItem)
                item->setSelected(true);
            else
                item->setSelected(false);
        }
    }
}

void ContainerSetting::onAddItem(QAction *action)
{
    int type = action->data().toInt();
    switch (type)
    {
    case TAB_CONFIG_GUIDE_ITEM_TYP_NETWORK_CARD:
    {
        GuideItem *item = createGuideItem(ui->listwidget_base_config,
                                          tr(NETWORK_CARD),
                                          GUIDE_ITEM_TYPE_NORMAL,
                                          ":/images/container-net-card.svg");
        m_baseItemMap.append(item);
        updateRemovableItem(NETWORK_CARD);
        NetworkConfPage *networkConfPage = new NetworkConfPage(this);
        m_baseConfStack->addWidget(networkConfPage);
        break;
    }
    default:
        break;
    }
}

void ContainerSetting::onDelItem()
{
    KLOG_INFO() << "onDelItem";
    GuideItem *guideItem = qobject_cast<GuideItem *>(sender());
    auto ret = MessageDialog::message(tr("Delete Network Card"),
                                      tr("Are you sure you want to delete the network card?"),
                                      tr("It can't be recovered after deletion.Are you sure you want to continue?"),
                                      ":/images/warning.svg",
                                      MessageDialog::StandardButton::Yes | MessageDialog::StandardButton::Cancel);
    if (ret == MessageDialog::StandardButton::Yes)
    {
        int row = 0;
        while (row < ui->listwidget_base_config->count() && m_netWorkCount > 1)
        {
            QListWidgetItem *item = ui->listwidget_base_config->item(row);
            if (ui->listwidget_base_config->itemWidget(item) == guideItem)
            {
                QListWidgetItem *delItem = ui->listwidget_base_config->takeItem(ui->listwidget_base_config->row(item));
                m_baseItemMap.removeAt(row);
                auto page = m_baseConfStack->widget(row);
                m_baseConfStack->removeWidget(page);
                m_netWorkCount--;
                updateRemovableItem(NETWORK_CARD);

                delete page;
                page = nullptr;
                delete delItem;
                delItem = nullptr;

                break;
            }
            row++;
        }
    }
    else
        KLOG_INFO() << "cancel";
}

void ContainerSetting::onConfirm()
{
    switch (m_type)
    {
    case CONTAINER_SETTING_TYPE_CONTAINER_CREATE:
        createContainer();
        break;
    case CONTAINER_SETTING_TYPE_CONTAINER_EDIT:
        updateContainer();
    default:
        break;
    }
}

void ContainerSetting::onNodeSelectedChanged(QString newStr)
{
    auto networkPage = qobject_cast<NetworkConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_NETWORK_CARD));
    networkPage->updateNetworkInfo(m_nodeInfo.key(newStr));
    getImageInfo(m_nodeInfo.key(newStr));
}

void ContainerSetting::getNodeListResult(const QPair<grpc::Status, node::ListReply> &reply)
{
    KLOG_INFO() << "getNodeListResult";
    if (reply.first.ok())
    {
        m_nodeInfo.clear();
        for (auto n : reply.second.nodes())
        {
            m_nodeInfo.insert(n.id(), QString("%1").arg(n.address().data()));
            ui->cb_node->addItem(QString("%1").arg(n.address().data()));
            m_totalCPU = n.status().cpu_stat().total();
        }
        if (!m_nodeInfo.isEmpty())
        {
            auto iter = m_nodeInfo.begin();
            onNodeSelectedChanged(iter.value());
            KLOG_INFO() << "total cpu = " << m_totalCPU;
        }
    }
}

void ContainerSetting::getCreateContainerResult(const QPair<grpc::Status, container::CreateReply> &reply)
{
    KLOG_INFO() << "getCreateContainerResult";
    if (reply.first.ok())
    {
        KLOG_INFO() << "create container successful!";
        emit sigUpdateContainer();
        close();
    }
    else
    {
        KLOG_DEBUG() << QString::fromStdString(reply.first.error_message());
        MessageDialog::message(tr("Create Container"),
                               tr("Create container failed!"),
                               tr("Error: ") + reply.first.error_message().data(),
                               ":/images/warning.svg",
                               MessageDialog::StandardButton::Ok);
    }
}

void ContainerSetting::getContainerInspectResult(const QPair<grpc::Status, container::InspectReply> &reply)
{
    KLOG_INFO() << "getContainerInspectResult";
    if (reply.first.ok())
    {
        //init ui
        auto &info = reply.second.info();
        ui->lineEdit_name->setText(info.name().data());
        KLOG_INFO() << info.name().data();

        auto &ctnCfg = reply.second.config();
        KLOG_INFO() << ctnCfg.image().data();
        if (m_labImage)
            m_labImage->setText(ctnCfg.image().data());

        if (ctnCfg.labels_size() > 0)
        {
            auto labels = ctnCfg.labels();
            if (labels.find(TagContainerDescription) != labels.end())
            {
                // display container description
                ui->lineEdit_describe->setText(labels[TagContainerDescription].data());
            }
        }

        auto host = reply.second.host_config();
        //cpu
        auto cpuPage = qobject_cast<CPUConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYPE_CPU));
        cpuPage->setCPUInfo(&host);

        //memory
        auto memoryPage = qobject_cast<MemoryConfPage *>(m_baseConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_MEMORY));
        memoryPage->setMemoryInfo(&host);

        //high-availability
        auto highAvailabilityPage = qobject_cast<HighAvailabilityPage *>(m_advancedConfStack->widget(TAB_CONFIG_GUIDE_ITEM_TYP_HIGH_AVAILABILITY));
        highAvailabilityPage->setRestartPolicy(&host);
    }
}

void ContainerSetting::getUpdateContainerResult(const QPair<grpc::Status, container::UpdateReply> &reply)
{
    if (reply.first.ok())
    {
        KLOG_INFO() << "update container successful!";
        sigUpdateContainer();
        close();
    }
    else
    {
        KLOG_DEBUG() << QString::fromStdString(reply.first.error_message());
        MessageDialog::message(tr("Update Container"),
                               tr("Update container failed!"),
                               tr("Error: %1").arg(reply.first.error_message().data()),
                               tr(":/images/warning.svg"),
                               MessageDialog::StandardButton::Ok);
    }
}

void ContainerSetting::getListImageFinishedResult(const QPair<grpc::Status, image::ListReply> &reply)
{
    KLOG_INFO() << "getListImageFinishedResult";
    if (reply.first.ok())
    {
        if (m_cbImage)
        {
            m_cbImage->clear();
            for (auto info : reply.second.images())
                m_cbImage->addItem(QString::fromStdString(info.name()));
        }
    }
    else
    {
        MessageDialog::message(tr("List Image"),
                               tr("Get image List failed!"),
                               tr("Error: %1").arg(reply.first.error_message().data()),
                               ":/images/warning.svg",
                               MessageDialog::StandardButton::Ok);
    }
}
