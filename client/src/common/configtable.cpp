#include "configtable.h"
#include <QDebug>
#include "ui_configtable.h"
#include<kiran-log/qt5-log-i.h>

ConfigTable::ConfigTable(ConfigTableType whichTable, QWidget *parent) : QWidget(parent),
                                                                        ui(new Ui::ConfigTable),
                                                                        m_ChooseTable(whichTable)
{
    ui->setupUi(this);
    m_pHeaderView = nullptr;
    m_pModel = nullptr;
    m_pDelegate = nullptr;
    initwindow();
    connect(m_pDelegate.get(), SIGNAL(sendSaveSig(int)), this, SLOT(saveRowSlot(int)));
    connect(m_pDelegate.get(), SIGNAL(sendEditSig(int)), this, SLOT(editRowSlot(int)));
    connect(m_pDelegate.get(), SIGNAL(sendAddSig(int)), this, SLOT(addRowSlot(int)));
    connect(m_pDelegate.get(), SIGNAL(sendDeleteSig(int)), this, SLOT(removeRowSlot(int)));
}

ConfigTable::~ConfigTable()
{
    delete ui;
}

void ConfigTable::initwindow()
{
    ui->tableView->verticalHeader()->setVisible(false);
    ui->tableView->horizontalHeader()->setMouseTracking(true);
    ui->tableView->setFocusPolicy(Qt::NoFocus);
    ui->tableView->setSelectionMode(QAbstractItemView::NoSelection);
    ui->tableView->setSelectionBehavior(QAbstractItemView::SelectRows);
    ui->tableView->setAlternatingRowColors(true);
    ui->tableView->setShowGrid(true);
    ui->tableView->setMouseTracking(true);
    ui->tableView->resizeColumnsToContents();
    ui->tableView->setFocusPolicy(Qt::NoFocus);
    m_pHeaderView.reset(new ConfigView(Qt::Horizontal, this));
    ui->tableView->setHorizontalHeader(m_pHeaderView.get());
    ui->tableView->horizontalHeader()->setStretchLastSection(true);

    QStringList head;

    if (m_ChooseTable)
        head << tr("Container path") << tr("Host path") << tr("Permission") << tr("Quick Actions");
    else
        head << tr("Name") << tr("Value") << tr("Permission") << tr("Quick Actions");

    ui->tableView->horizontalHeader()->setStyleSheet("QHeaderView::section, QTableCornerButton::section {padding:1px;border:none;background-color:WhiteSmoke;color:Black;}");
    m_pModel.reset(new ConfigModel(head));
    ui->tableView->setModel(m_pModel.get());

    qDebug() << __func__ << ",whichTable:" << m_ChooseTable;
    if (m_ChooseTable)
    {
        ui->tableView->setColumnWidth(0, 150);
        ui->tableView->setColumnWidth(1, 150);
        ui->tableView->setColumnWidth(2, 80);
    }
    else
    {
        ui->tableView->setColumnWidth(0, 190);
        ui->tableView->setColumnWidth(1, 190);
        ui->tableView->setColumnHidden(2, true);
    }

    m_pDelegate.reset(new ConfigDelegate(m_ChooseTable));
    ui->tableView->setItemDelegate(m_pDelegate.get());
    initTable();
}

void ConfigTable::initTable()
{
    QSharedPointer<ModelItem> pItem(new ModelItem());
    m_pModel->addModelRow(pItem);
    paintEditor(m_pModel->rowCount(QModelIndex()) - 1);
}

void ConfigTable::paintEditor(int curRow)
{
    Q_UNUSED(curRow);
    int RowCount = m_pModel->rowCount(QModelIndex());
    for (int i = 0; i < RowCount; ++i)
    {
        ui->tableView->openPersistentEditor(m_pModel->index(i, FIRST_COLUMN_HEADER));
        ui->tableView->openPersistentEditor(m_pModel->index(i, SECOND_COLUMN_HEADER));
        ui->tableView->openPersistentEditor(m_pModel->index(i, THIRD_COLUMN_HEADER));
        ui->tableView->openPersistentEditor(m_pModel->index(i, OPERATE_HEADER));
    }
}

QList<QSharedPointer<ModelItem> > ConfigTable::getAllData()
{
    //paintEditor();
    auto i = m_pModel->getAllModel();
    for (auto pItem : i)
    {
        KLOG_INFO() << __func__ << ",Var:" << pItem->m_firstColVal << ",Val:" << pItem->m_secondColVal << ",mode:" << pItem->m_thirdColVal;
    }

    return m_pModel->getAllModel();
}

void ConfigTable::saveRowSlot(int row)
{
    auto pItem = m_pModel->getModelByRow(row);
    qDebug() << __func__ << ", Var:" << pItem->m_firstColVal << ",Val:" << pItem->m_secondColVal << ",mode:" << pItem->m_thirdColVal;
}

void ConfigTable::editRowSlot(int row)
{
    auto pItem = m_pModel->getModelByRow(row);
    qDebug() << __func__ << ", Var:" << pItem->m_firstColVal << ",Val:" << pItem->m_secondColVal << ",mode:" << pItem->m_thirdColVal;
}

void ConfigTable::addRowSlot(int row)
{
    QSharedPointer<ModelItem> pItem(new ModelItem());
    m_pModel->insertModelByRow(row + 1, pItem);
    paintEditor(row);
}

void ConfigTable::removeRowSlot(int row)
{
    m_pModel->deleteModelByRow(row);
    paintEditor(row);
}
