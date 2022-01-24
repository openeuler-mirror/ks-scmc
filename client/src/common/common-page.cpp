#include "common-page.h"
#include <kiran-log/qt5-log-i.h>
#include <QHBoxLayout>
#include <QTimer>
#include <iostream>
#include "common/button-delegate.h"
#include "common/header-view.h"
#include "ui_common-page.h"

using namespace std;

#define TIMEOUT 200
CommonPage::CommonPage(QWidget *parent) : QWidget(parent),
                                          ui(new Ui::CommonPage)
{
    ui->setupUi(this);
    initUI();
    m_timer = new QTimer(this);
    connect(m_timer, &QTimer::timeout,
            [this] {
                search();
                m_timer->stop();
            });
}

CommonPage::~CommonPage()
{
    delete ui;
    if (m_timer)
    {
        delete m_timer;
        m_timer = nullptr;
    }
}

void CommonPage::setBusy(bool status)
{
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
    KLOG_INFO() << "setOpBtnEnabled" << enabled;
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

QStandardItem *CommonPage::getRowItem(int row)
{
    return m_model->item(row, 0);
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

void CommonPage::initUI()
{
    //ui->tableView->installEventFilter(this);
    ui->lineEdit_search->setPlaceholderText(tr("Please enter the keyword"));
    ui->btn_refresh->setIcon(QIcon(":/images/refresh.svg"));

    QHBoxLayout *layout = new QHBoxLayout(ui->lineEdit_search);
    layout->setMargin(0);
    layout->setContentsMargins(10, 0, 10, 0);

    QPushButton *btn_search = new QPushButton(ui->lineEdit_search);
    btn_search->setObjectName("btn_search");
    btn_search->setFixedSize(QSize(16, 16));
    btn_search->setIcon(QIcon(":/images/search.svg"));
    btn_search->setStyleSheet("#btn_search{background:transparent;}");
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
                m_timer->start(TIMEOUT);
            });
}

void CommonPage::adjustTableSize()
{
    int height = 0;
    height = m_model->rowCount() * 50 + 50 + 20;  // row height+ header height + space
    ui->tableView->setFixedHeight(height);
    emit sigTableHeightChanged(height);
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
