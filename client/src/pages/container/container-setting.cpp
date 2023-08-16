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

ContainerSetting::ContainerSetting(QWidget *parent) : QWidget(parent),
                                                      ui(new Ui::ContainerSetting),
                                                      m_netWorkCount(0)
{
    ui->setupUi(this);
    initUI();
    setAttribute(Qt::WA_DeleteOnClose);
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

    initBaseConfPages();
    initAdvancedConfPages();

    QList<QPair<QString, QString>> baseConfItemInfo = {{tr(CPU), ":/images/container-cpu.svg"},
                                                       {tr(MEMORY), ":/images/container-memory.svg"},
                                                       {tr(NETWORK_CARD), ":/images/container-net-card.svg"}};
    for (int i = 0; i < baseConfItemInfo.count(); i++)
    {
        QString name = baseConfItemInfo.at(i).first;
        GuideItem *item = createGuideItem(ui->listwidget_base_config,
                                          name,
                                          GUIDE_ITEM_TYPE_NORMAL,
                                          baseConfItemInfo.at(i).second);
        m_baseItemMap.append(item);
    }

    QList<QPair<QString, QString>> advancedConfItemInfo = {{tr(ENVS), ":/images/container-env.png"},
                                                           {tr(GRAPHIC), ":/images/audit-center.svg"},
                                                           {tr(VOLUMES), ":/images/container-volumes.png"},
                                                           {tr(HIGH_AVAILABILITY), ":/images/container-high-avail.png"}};
    for (int i = 0; i < advancedConfItemInfo.count(); i++)
    {
        QString name = advancedConfItemInfo.at(i).first;
        GuideItem *item = createGuideItem(ui->listWidget_advanced_config,
                                          name,
                                          GUIDE_ITEM_TYPE_NORMAL,
                                          advancedConfItemInfo.at(i).second);
        m_advancedItemMap.append(item);
    }

    connect(ui->listwidget_base_config, &QListWidget::itemClicked, this, &ContainerSetting::onItemClicked);
    connect(ui->listWidget_advanced_config, &QListWidget::itemClicked, this, &ContainerSetting::onItemClicked);
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
        connect(customItem, &GuideItem::sigDeleteItem, this, &ContainerSetting::popupMessageDialog);
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
    CPUConfPage *cpuInfoPage = new CPUConfPage(ui->tab_base_config);
    m_baseConfStack->addWidget(cpuInfoPage);

    MemoryConfPage *memoryConfPage = new MemoryConfPage(ui->tab_base_config);
    m_baseConfStack->addWidget(memoryConfPage);

    NetworkConfPage *networkConfPage = new NetworkConfPage(ui->tab_base_config);
    m_baseConfStack->addWidget(networkConfPage);

    QList<QComboBox *> comboboxs = memoryConfPage->findChildren<QComboBox *>();
    foreach (QComboBox *cb, comboboxs)
    {
        cb->addItems(QStringList() << "MB"
                                   << "GB");
    }
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

QStringList ContainerSetting::getNodes()
{
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

void ContainerSetting::popupMessageDialog()
{
    GuideItem *guideItem = qobject_cast<GuideItem *>(sender());
    MessageDialog *messageDialog = new MessageDialog(guideItem, this);
    messageDialog->setAttribute(Qt::WA_DeleteOnClose);
    messageDialog->setTitle(tr("Delete Network Card"));
    messageDialog->setSummary(tr("Are you sure you want to delete the network card?"));
    messageDialog->setBody(tr("It can't be recovered after deletion.Are you sure you want to continue?"));
    messageDialog->setIcon(":/images/warning.png");
    messageDialog->setWidth(600);
    messageDialog->setModal(true);
    messageDialog->show();
    connect(messageDialog, &MessageDialog::sigConfirm, this, &ContainerSetting::onDelItem);
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

void ContainerSetting::onDelItem(QWidget *sender)
{
    KLOG_INFO() << "onDelItem";
    GuideItem *guideItem = qobject_cast<GuideItem *>(sender);
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

            delete page;
            page = nullptr;
            delete delItem;
            delItem = nullptr;
            m_netWorkCount--;
            updateRemovableItem(NETWORK_CARD);
            break;
        }
        row++;
    }
}
