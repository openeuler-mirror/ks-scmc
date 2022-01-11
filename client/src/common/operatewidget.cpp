#include "operatewidget.h"
#include "ui_operatewidget.h"

OperateWidget::OperateWidget(int curRow, int page, QWidget *parent) : QWidget(parent),
                                                                      ui(new Ui::OperateWidget),
                                                                      m_CurRow(curRow)
{
    ui->setupUi(this);

    initControl();
    initConnections();
    ui->stackedWidget->setCurrentIndex(page);
}

OperateWidget::~OperateWidget()
{
    delete ui;
}

int OperateWidget::getCurRow()
{
    return m_CurRow;
}

int OperateWidget::getCurPage()
{
    return ui->stackedWidget->currentIndex();
}

void OperateWidget::setPage(int page)
{
    ui->stackedWidget->setCurrentIndex(page);
}

void OperateWidget::initControl()
{
    ui->m_pBtnSave->setText(tr("Save"));
    ui->m_pBtnEdit->setFocusPolicy(Qt::NoFocus);
    ui->m_pBtnEdit->setCursor(Qt::PointingHandCursor);
    ui->m_pBtnEdit->setIcon(QIcon(":/images/edit_item.svg"));
    ui->m_pBtnAdd->setFocusPolicy(Qt::NoFocus);
    ui->m_pBtnAdd->setCursor(Qt::PointingHandCursor);
    ui->m_pBtnAdd->setIcon(QIcon(":/images/add_item.svg"));
    ui->m_pBtnDel->setFocusPolicy(Qt::NoFocus);
    ui->m_pBtnDel->setCursor(Qt::PointingHandCursor);
    ui->m_pBtnDel->setIcon(QIcon(":/images/delete_item.svg"));
}

void OperateWidget::initConnections()
{
    connect(ui->m_pBtnSave, SIGNAL(clicked()), this, SLOT(sendSave()));
    connect(ui->m_pBtnEdit, SIGNAL(clicked()), this, SLOT(sendEdit()));
    connect(ui->m_pBtnAdd, SIGNAL(clicked()), this, SLOT(sendAdd()));
    connect(ui->m_pBtnDel, SIGNAL(clicked()), this, SLOT(sendDelete()));
}

void OperateWidget::sendSave()
{
    emit sendSaveSig(this);
}

void OperateWidget::sendEdit()
{
    ui->stackedWidget->setCurrentWidget(ui->page);
    emit sendEditSig(this);
}

void OperateWidget::sendAdd()
{
    emit sendAddSig(this);
}

void OperateWidget::sendDelete()
{
    emit sendDeleteSig(this);
}
