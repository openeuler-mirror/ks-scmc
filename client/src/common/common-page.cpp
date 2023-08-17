#include "common-page.h"
#include <kiran-log/qt5-log-i.h>
#include <QHBoxLayout>
#include <QPainter>
#include <QTime>
#include <QTimer>
#include <iostream>
#include "common/button-delegate.h"
#include "common/header-view.h"
#include "common/mask-widget.h"
#include "ui_common-page.h"

using namespace std;

#define TIMEOUT 200
CommonPage::CommonPage(QWidget *parent) : QWidget(parent),
                                          ui(new Ui::CommonPage),
                                          m_searchTimer(nullptr),
                                          m_refreshBtnTimer(nullptr),
                                          m_maskWidget(nullptr)
{
    ui->setupUi(this);
    initUI();

    m_maskWidget = new MaskWidget(this);
    m_maskWidget->setFixedSize(this->size());  //设置窗口大小
    this->stackUnder(qobject_cast<QWidget *>(m_maskWidget));

    m_searchTimer = new QTimer(this);
    connect(m_searchTimer, &QTimer::timeout,
            [this] {
                search();
                m_searchTimer->stop();
            });
    m_refreshBtnTimer = new QTimer(this);
    connect(m_refreshBtnTimer, &QTimer::timeout, this, &CommonPage::onRefreshTimeout);
}

CommonPage::~CommonPage()
{
    delete ui;
    if (m_searchTimer)
    {
        delete m_searchTimer;
        m_searchTimer = nullptr;
    }
    if (m_refreshBtnTimer)
    {
        delete m_refreshBtnTimer;
        m_refreshBtnTimer = nullptr;
    }
}

void CommonPage::setBusy(bool status)
{
    m_maskWidget->setMaskVisible(status);
    setOpBtnEnabled(!status);
}

void CommonPage::clearTable()
{
    KLOG_INFO() << "pre" << m_model->rowCount();
    m_model->removeRows(0, m_model->rowCount());
    KLOG_INFO() << "current" << m_model->rowCount();
}

void CommonPage::addOperationButton(QToolButton *btn)
{
    ui->hLayout_OpBtns->addWidget(btn, Qt::AlignLeft);
}

void CommonPage::addOperationButtons(QList<QPushButton *> opBtns)
{
    foreach (QPushButton *btn, opBtns)
    {
        ui->hLayout_OpBtns->addWidget(btn, Qt::AlignLeft);
    }
}

void CommonPage::setOpBtnEnabled(bool enabled)
{
    for (int i = 0; i < ui->hLayout_OpBtns->count(); i++)
    {
        QAbstractButton *btn = qobject_cast<QAbstractButton *>(ui->hLayout_OpBtns->itemAt(i)->widget());
        btn->setEnabled(enabled);
    }
}

void CommonPage::setTableRowNum(int num)
{
    m_model->setRowCount(num);
}

void CommonPage::setTableColNum(int num)
{
    m_model->setColumnCount(num);
}

void CommonPage::setTableItem(int row, int col, QStandardItem *item)
{
    m_model->setItem(row, col, item);
    adjustTableSize();
}

void CommonPage::setTableItems(int row, int col, QList<QStandardItem *> items)
{
    for (int i = col, j = 0; i < items.size() + col; i++, j++)
        m_model->setItem(row, i, items.at(j));
    adjustTableSize();
}

void CommonPage::setTableActions(int col, QStringList actionIcons)
{
    //设置表中操作按钮代理
    QMap<int, QString> btnInfo;
    for (int i = 0; i < actionIcons.size(); i++)
    {
        btnInfo.insert(i, actionIcons.at(i));
    }
    ButtonDelegate *btnDelegate = new ButtonDelegate(btnInfo, this);
    ui->tableView->setItemDelegateForColumn(col, btnDelegate);
    m_actionCol = col;

    connect(btnDelegate, &ButtonDelegate::sigMonitor, this, &CommonPage::onMonitor);
    connect(btnDelegate, &ButtonDelegate::sigEdit, this, &CommonPage::onEdit);
    connect(btnDelegate, &ButtonDelegate::sigTerminal, this, &CommonPage::onTerminal);
    connect(btnDelegate, &ButtonDelegate::sigActRun, this, &CommonPage::onActRun);
    connect(btnDelegate, &ButtonDelegate::sigActStop, this, &CommonPage::onActStop);
    connect(btnDelegate, &ButtonDelegate::sigActRestart, this, &CommonPage::onActRestart);
}

void CommonPage::setSortableCol(QList<int> cols)
{
    m_headerView->setSortableCols(cols);
}

void CommonPage::setHeaderSections(QStringList names)
{
    //插入表头数据
    for (int i = 0; i < names.size(); i++)
    {
        QStandardItem *headItem = new QStandardItem(names.at(i));
        m_model->setHorizontalHeaderItem(i, headItem);
    }
    ui->tableView->horizontalHeader()->setSectionResizeMode(QHeaderView::Interactive);
    ui->tableView->horizontalHeader()->setSectionResizeMode(1, QHeaderView::Fixed);

    //设置列宽度
    for (int i = 0; i < names.size(); i++)
    {
        ui->tableView->setColumnWidth(i, 150);
    }
    ui->tableView->setColumnWidth(0, 300);
    ui->tableView->setColumnWidth(1, 150);
}

void CommonPage::setTableDefaultContent(QList<int> actionCol, QString text)
{
    m_model->removeRows(0, m_model->rowCount());
    for (int i = 0; i < m_model->columnCount(); i++)
    {
        if (!actionCol.contains(i))
        {
            QStandardItem *item = new QStandardItem(text);
            item->setTextAlignment(Qt::AlignCenter);
            m_model->setItem(0, i, item);
        }
    }
}

void CommonPage::clearText()
{
    ui->label_search_tips->clear();
    ui->lineEdit_search->clear();
}

int CommonPage::getTableRowCount()
{
    return m_model->rowCount();
}

QStandardItem *CommonPage::getItem(int row, int col)
{
    return m_model->item(row, col);
}

QList<QMap<QString, QVariant>> CommonPage::getCheckedItemInfo(int col)
{
    QList<QMap<QString, QVariant>> checkedItemInfo;  //containerId,nodeId
    for (int i = 0; i < m_model->rowCount(); i++)
    {
        auto item = m_model->item(i, col);
        if (item->checkState() == Qt::CheckState::Checked)
        {
            QMap<QString, QVariant> idMap = item->data().value<QMap<QString, QVariant>>();
            checkedItemInfo.append(idMap);
        }
    }
    return checkedItemInfo;
}

void CommonPage::sleep(int sec)
{
    QTime dieTime = QTime::currentTime().addSecs(sec);
    while (QTime::currentTime() < dieTime)
        QCoreApplication::processEvents(QEventLoop::AllEvents, 100);
}

void CommonPage::initUI()
{
    ui->lineEdit_search->setPlaceholderText(tr("Please enter the keyword"));
    ui->btn_refresh->setIcon(QIcon(":/images/refresh.svg"));
    ui->btn_refresh->installEventFilter(this);

    QHBoxLayout *layout = new QHBoxLayout(ui->lineEdit_search);
    layout->setMargin(0);
    layout->setContentsMargins(10, 0, 10, 0);

    QPushButton *btn_search = new QPushButton(ui->lineEdit_search);
    btn_search->setObjectName("btn_search");
    btn_search->setFixedSize(QSize(16, 16));
    btn_search->setIcon(QIcon(":/images/search.svg"));
    btn_search->setStyleSheet("#btn_search{background:#ffffff;border:none;}"
                              "#btn_search:focus{outline:none;}");
    btn_search->setCursor(Qt::PointingHandCursor);
    layout->addStretch();
    layout->addWidget(btn_search);
    ui->lineEdit_search->setTextMargins(10, 0, btn_search->width() + 10, 0);

    m_model = new QStandardItemModel(this);
    ui->tableView->setModel(m_model);

    //设置表头
    m_headerView = new HeaderView(true, ui->tableView);
    m_headerView->setStretchLastSection(true);
    m_headerView->setStyleSheet("alignment: left;");
    ui->tableView->setHorizontalHeader(m_headerView);
    //隐藏列表头
    ui->tableView->verticalHeader()->setVisible(false);
    ui->tableView->verticalHeader()->setDefaultSectionSize(50);

    //设置表的其他属性
    ui->tableView->setMouseTracking(true);
    ui->tableView->setSelectionMode(QAbstractItemView::NoSelection);
    ui->tableView->setEditTriggers(QAbstractItemView::NoEditTriggers);
    ui->tableView->setSortingEnabled(true);
    ui->tableView->setFocusPolicy(Qt::NoFocus);

    connect(btn_search, &QPushButton::clicked, this, &CommonPage::search);
    connect(m_headerView, &HeaderView::ckbToggled, this, &CommonPage::onHeaderCkbTog);
    connect(ui->btn_refresh, &QToolButton::clicked, this, &CommonPage::refresh);
    connect(ui->lineEdit_search, &QLineEdit::textChanged,
            [this](QString text) {
                m_searchTimer->start(TIMEOUT);
            });
}

void CommonPage::adjustTableSize()
{
    int height = 0;
    height = m_model->rowCount() * 50 + 50 + 20;  // row height+ header height + space
    ui->tableView->setFixedHeight(height);
    emit sigTableHeightChanged(height);
}

bool CommonPage::eventFilter(QObject *watched, QEvent *event)
{
    if (watched == ui->btn_refresh && event->type() == QEvent::HoverEnter)
    {
        ui->btn_refresh->setIcon(QIcon(":/images/refresh-hover.svg"));
        return true;
    }
    else if (watched == ui->btn_refresh && event->type() == QEvent::HoverLeave)
    {
        ui->btn_refresh->setIcon(QIcon(":/images/refresh.svg"));
        return true;
    }
    return false;
}

void CommonPage::resizeEvent(QResizeEvent *event)
{
    if (event)
    {
    }  //消除警告提示

    if (m_maskWidget != nullptr)
    {
        m_maskWidget->setAutoFillBackground(true);
        QPalette pal = m_maskWidget->palette();
        pal.setColor(QPalette::Background, QColor(0x00, 0x00, 0x00, 0x20));
        m_maskWidget->setPalette(pal);
        m_maskWidget->setFixedSize(this->size());
    }
}

void CommonPage::onMonitor(int row)
{
    KLOG_INFO() << "CommonPage::onMonitor" << row;
    emit sigMonitor(row);
}

void CommonPage::onTerminal(int row)
{
    KLOG_INFO() << "CommonPage::onTerminal" << row;
    emit sigTerminal(row);
}

void CommonPage::onEdit(int row)
{
    KLOG_INFO() << "CommonPage::onEdit" << row;
    emit sigEdit(row);
}

void CommonPage::onActRun(QModelIndex index)
{
    KLOG_INFO() << index.row();
    emit sigRun(index);
}

void CommonPage::onActStop(QModelIndex index)
{
    KLOG_INFO() << index.row();
    emit sigStop(index);
}

void CommonPage::onActRestart(QModelIndex index)
{
    KLOG_INFO() << index.row();
    emit sigRestart(index);
}

void CommonPage::onRefreshTimeout()
{
    static int count = 0;
    count++;
    QPixmap pix(":/images/refresh-hover.svg");
    static int rat = 0;
    rat = rat >= 180 ? 30 : rat + 30;
    cout << rat << endl;
    int imageWidth = pix.width();
    int imageHeight = pix.height();
    QPixmap temp(pix.size());
    temp.fill(Qt::transparent);
    QPainter painter(&temp);
    painter.setRenderHint(QPainter::SmoothPixmapTransform, true);
    painter.translate(imageWidth / 2, imageHeight / 2);        //让图片的中心作为旋转的中心
    painter.rotate(rat);                                       //顺时针旋转90度
    painter.translate(-(imageWidth / 2), -(imageHeight / 2));  //使原点复原
    painter.drawPixmap(0, 0, pix);
    painter.end();
    ui->btn_refresh->setIcon(QIcon(temp));

    if (count == 6)
    {
        m_refreshBtnTimer->stop();
        ui->btn_refresh->setIcon(QIcon(":/images/refresh.svg"));
        count = 0;
    }
}

void CommonPage::search()
{
    KLOG_INFO() << "search....";
    auto resultCount = 0;
    QString text = ui->lineEdit_search->text();
    if (text.isEmpty())
        updateInfo();

    else
    {
        //show keyword row
        int rowCounts = m_model->rowCount();
        for (int i = 0; i < rowCounts; i++)
        {
            QStandardItem *item = m_model->item(i, 0);
            if (!item->text().contains(text))
            {
                ui->tableView->setRowHidden(i, true);
            }
            else
            {
                ui->tableView->showRow(i);
                resultCount++;
            }
        }
        if (resultCount == 0)
        {
            ui->label_search_tips->setText(tr("No search results were found!"));
            ui->tableView->setFixedHeight(120);
            setOpBtnEnabled(false);
            return;
        }
        //sort
        ui->tableView->sortByColumn(0);
        ui->label_search_tips->clear();
        setOpBtnEnabled(true);
        adjustTableSize();
    }
}

void CommonPage::refresh()
{
    m_refreshBtnTimer->start(50);
    //更新列表信息
    updateInfo();
}

void CommonPage::onHeaderCkbTog(bool toggled)
{
    int rowCounts = m_model->rowCount();
    KLOG_INFO() << "onHeaderCkbTog" << rowCounts;
    for (int i = 0; i < rowCounts; i++)
    {
        QStandardItem *item = m_model->item(i, 0);
        if (item)
        {
            if (toggled)
                item->setCheckState(Qt::Checked);
            else
                item->setCheckState(Qt::Unchecked);
        }
    }
}
