#include "common-page.h"
#include <kiran-log/qt5-log-i.h>
#include <iostream>
#include "common/button-delegate.h"
#include "common/header-view.h"
#include "ui_common-page.h"

using namespace std;
CommonPage::CommonPage(QWidget *parent) : QWidget(parent),
                                          ui(new Ui::CommonPage)
{
    ui->setupUi(this);
    initUI();
    connect(ui->lineEdit_search, &QLineEdit::returnPressed, this, &CommonPage::search);
}

CommonPage::~CommonPage()
{
    delete ui;
}

void CommonPage::setBusy(bool status)
{
}

void CommonPage::clearTable()
{
    m_model->removeRows(0, m_model->rowCount());
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

void CommonPage::setTableRowNum(int num)
{
    m_model->setRowCount(num);
}

void CommonPage::setTableColNum(int num)
{
    m_model->setColumnCount(num);
}

void CommonPage::setTableItem(int col, int row, QStandardItem *item)
{
    m_model->setItem(row, col, item);
}

void CommonPage::setTableItems(int row, QList<QStandardItem *> items)
{
    for (int i = 0; i < items.size(); i++)
    {
        if (i == 0)
        {
            m_model->setItem(row, i, items.at(i));
        }
        else
        {
            m_model->setItem(row, i + 1, items.at(i));
        }
    }
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

QList<QMap<QString, QVariant>> CommonPage::getCheckedRowInfo()
{
    return m_checkedRowInfo;
}

void CommonPage::initUI()
{
    ui->lineEdit_search->setPlaceholderText(tr("Please enter the keyword"));
    ui->btn_refresh->setIcon(QIcon(":/images/refresh.svg"));

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

    connect(m_headerView, &HeaderView::ckbToggled, this, &CommonPage::onHeaderCkbTog);
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
    QString text = ui->lineEdit_search->text();
    cout << "search text" << endl;
    //show keyword row
    //sort
    updateInfo(text);
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
