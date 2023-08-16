#include "memory-conf-page.h"
#include "ui_memory-conf-page.h"

MemoryConfPage::MemoryConfPage(QWidget *parent) : QWidget(parent),
                                                  ui(new Ui::MemoryConfPage)
{
    ui->setupUi(this);
}

MemoryConfPage::~MemoryConfPage()
{
    delete ui;
}

void MemoryConfPage::setMemoryInfo(MemoryInfo memoryInfo)
{
    ui->lineEdit_soft_limit->setText(QString("%1").arg(memoryInfo.softLimit));
    ui->lineEdit_max_limit->setText(QString("%1").arg(memoryInfo.maxLimit));
}

MemoryInfo MemoryConfPage::getMemoryInfo()
{
    MemoryInfo memoryInfo;
    memoryInfo.softLimit = ui->lineEdit_soft_limit->text().toLongLong();
    memoryInfo.maxLimit = ui->lineEdit_max_limit->text().toLongLong();
}
