#include "main-window.h"
#include <QDebug>
#include <QIcon>
#include <QPainter>
#include <iostream>
#include "./ui_main-window.h"
#include "common/common-page.h"
#include "common/guide-item.h"
#include "common/info-worker.h"
#include "pages/container/container-list.h"
#include "pages/node/node-list.h"

std::string g_server_addr = "10.200.12.181:10050";

MainWindow::MainWindow(QWidget* parent)
    : QWidget(parent), ui(new Ui::MainWindow)
{
    ui->setupUi(this);

    QStringList args = qApp->arguments();
    if (args.size() > 1)
    {
        g_server_addr = args[1].toStdString();
        printf("[%s][%d] g_server_addr:%s\n", __func__, __LINE__, g_server_addr.data());
    }

    initUI();
}

MainWindow::~MainWindow()
{
    delete ui;
}

void MainWindow::onItemClicked(QListWidgetItem* currItem, QListWidgetItem* preItem)
{
    //侧边栏展开与收缩
    GuideItem* guideItem = qobject_cast<GuideItem*>(ui->listWidget->itemWidget(currItem));
    if (guideItem->getItemType() != GUIDE_ITEM_TYPE_GROUP)
    {
        guideItem->setSelected(true);
        foreach (GuideItem* item, m_pageItems)
        {
            if (item != guideItem)
            {
                item->setSelected(false);
            }
        }
    }
    if (m_groupMap.contains(currItem))
    {
        if (m_isShowMap.value(currItem))  //hide
        {
            guideItem->setArrow(true);
            QList<QListWidgetItem*> subItems = m_groupMap.value(currItem);
            foreach (QListWidgetItem* subItem, subItems)
            {
                subItem->setHidden(true);
            }
            m_isShowMap.insert(currItem, false);
        }
        else  //show
        {
            guideItem->setArrow(false);
            QList<QListWidgetItem*> subItems = m_groupMap.value(currItem);
            foreach (QListWidgetItem* subItem, subItems)
            {
                subItem->setHidden(false);
            }
            m_isShowMap.insert(currItem, true);
        }
    }
    int currenRow = ui->listWidget->row(currItem);
    std::cout << currenRow << std::endl;

    if (!m_pageMap.value(currenRow))
        return;
    m_pageMap[currenRow]->updateInfo();
    m_stackedWidget->setCurrentWidget(m_pageMap.value(currenRow));
}

void MainWindow::paintEvent(QPaintEvent* event)
{
    Q_UNUSED(event);
    QStyleOption opt;
    opt.init(this);
    QPainter p(this);
    style()->drawPrimitive(QStyle::PE_Widget, &opt, &p, this);
}

void MainWindow::initUI()
{
    this->setWindowTitle(tr("KylinSec security Container magic Cube"));
    ui->btn_menu->setIcon(QIcon(":/images/menu.svg"));
    ui->label_name->setText("Admin");

    m_stackedWidget = new QStackedWidget(this);
    m_stackedWidget->setObjectName("stackedWidget");
    QLayout* layout = this->layout();
    layout->addWidget(m_stackedWidget);

    const QMap<GUIDE_ITEM, QString> pageMap = {
        {GUIDE_ITEM_CONTAINER_LIST, tr("Container List")},
        {GUIDE_ITEM_NODE_MANAGER, tr("Node List")}};
    for (auto iter = pageMap.begin(); iter != pageMap.end(); iter++)
    {
        GUIDE_ITEM itemEnum = iter.key();
        QString desc = iter.value();
        qInfo() << desc << itemEnum;

        auto subPage = createSubPage(itemEnum);
        if (!subPage)
        {
            qWarning() << "sub page is null,ignore!";
            continue;
        }
        m_stackedWidget->addWidget(subPage);
        m_pageMap[itemEnum] = subPage;
    }

    QListWidgetItem* homeItem = createGuideItem(tr("Home Page"), GUIDE_ITEM_TYPE_NORMAL, ":/images/home-page.png");
    QListWidgetItem* auditCenter = createGuideItem(tr("Audit Center"), GUIDE_ITEM_TYPE_GROUP, ":/images/audit-center.svg");
    QListWidgetItem* auditApplyList = createGuideItem(tr("Audit Apply List"), GUIDE_ITEM_TYPE_SUB);
    QListWidgetItem* auditWarningList = createGuideItem(tr("Audit Warning List"), GUIDE_ITEM_TYPE_SUB);
    QListWidgetItem* auditLogList = createGuideItem(tr("Audit Log List"), GUIDE_ITEM_TYPE_SUB);
    QList<QListWidgetItem*> auditSubItems = {auditApplyList, auditWarningList, auditLogList};
    m_groupMap.insert(auditCenter, auditSubItems);
    ///TODO: m_isShowMap.insert(auditCenter, false);
    m_isShowMap.insert(auditCenter, true);

    QListWidgetItem* containerManager = createGuideItem(tr("Container Manager"), GUIDE_ITEM_TYPE_GROUP, ":/images/container-manager.svg");
    QListWidgetItem* containerList = createGuideItem(tr("Container List"), GUIDE_ITEM_TYPE_SUB);
    QListWidgetItem* containerTemplate = createGuideItem(tr("Container Template"), GUIDE_ITEM_TYPE_SUB);
    QList<QListWidgetItem*> containerSubItems = {containerList, containerTemplate};
    m_groupMap.insert(containerManager, containerSubItems);
    m_isShowMap.insert(containerManager, false);

    QListWidgetItem* imageManager = createGuideItem(tr("Image Manager"), GUIDE_ITEM_TYPE_NORMAL, ":/images/image-manager.svg");
    QListWidgetItem* nodeManager = createGuideItem(tr("Node Manager"), GUIDE_ITEM_TYPE_NORMAL, ":/images/node-manager.svg");

    //show first
    GuideItem* guideItem = qobject_cast<GuideItem*>(ui->listWidget->itemWidget(containerManager));
    guideItem->setArrow(false);

    GuideItem* item = qobject_cast<GuideItem*>(ui->listWidget->itemWidget(containerList));
    item->setSelected(true);

    foreach (QListWidgetItem* subItem, containerSubItems)
    {
        subItem->setHidden(false);
    }
    ///TODO:set current widget to home
    m_stackedWidget->setCurrentWidget(m_pageMap.value(GUIDE_ITEM_CONTAINER_LIST));
    ui->listWidget->setCurrentRow(GUIDE_ITEM_CONTAINER_LIST);
    m_pageMap[GUIDE_ITEM_CONTAINER_LIST]->updateInfo();

    connect(ui->listWidget, &QListWidget::currentItemChanged, this, &MainWindow::onItemClicked);
}

CommonPage* MainWindow::createSubPage(GUIDE_ITEM itemEnum)
{
    CommonPage* page = nullptr;

    switch (itemEnum)
    {
    case GUIDE_ITEM_CONTAINER_LIST:
    {
        page = new ContainerList();
        break;
    }
    case GUIDE_ITEM_CONTAINER_TEMPLATE:
    {
    }
    case GUIDE_ITEM_NODE_MANAGER:
    {
        page = new NodeList();
        break;
    }
    default:
        break;
    }
    return page;
}

QListWidgetItem* MainWindow::createGuideItem(QString text, int type, QString icon)
{
    QListWidgetItem* newItem = nullptr;
    GuideItem* customItem = nullptr;

    newItem = new QListWidgetItem(ui->listWidget);

    customItem = new GuideItem(text, icon, type, ui->listWidget);

    ui->listWidget->addItem(newItem);
    ui->listWidget->setItemWidget(newItem, customItem);
    newItem->setTextAlignment(Qt::AlignVCenter);

    newItem->setSizeHint(QSize(220, 60));
    if (type == GUIDE_ITEM_TYPE_SUB)
    {
        newItem->setSizeHint(QSize(220, 50));
        newItem->setHidden(true);
        m_pageItems.append(customItem);
    }
    else if (type == GUIDE_ITEM_TYPE_NORMAL)
    {
        m_pageItems.append(customItem);
    }
    else
        customItem->setArrow(true);

    return newItem;
}
