#include "cpu-conf-page.h"
#include "ui_cpu-conf-page.h"

CPUConfPage::CPUConfPage(QWidget *parent) : QWidget(parent),
                                            ui(new Ui::CPUConfPage)
{
    ui->setupUi(this);
}

CPUConfPage::~CPUConfPage()
{
    delete ui;
}

void CPUConfPage::setCPUInfo(CPUInfo cpuInfo)
{
    ui->lineEdit_cpu_core->setText(QString("%1").arg(cpuInfo.totalCore));
    ui->lineEdit_sche_pri->setText(QString("%1").arg(cpuInfo.schedulingPriority));
}

CPUInfo CPUConfPage::getCPUInfo()
{
    CPUInfo cpuInfo;
    cpuInfo.totalCore = ui->lineEdit_cpu_core->text().toInt();
    cpuInfo.schedulingPriority = ui->lineEdit_sche_pri->text().toInt();
}
