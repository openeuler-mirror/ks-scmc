#include "cpu-conf-page.h"
#include <kiran-log/qt5-log-i.h>
#include <QValidator>
#include "ui_cpu-conf-page.h"
CPUConfPage::CPUConfPage(double totalCPU, QWidget* parent) : QWidget(parent),
                                                             ui(new Ui::CPUConfPage),
                                                             m_totalCPU(totalCPU)
{
    ui->setupUi(this);
    QValidator* intValidator = new QIntValidator(2, 262144, this);
    ui->lineEdit_sche_pri->setValidator(intValidator);

    QDoubleValidator* doubleValidator = new QDoubleValidator(0.01, totalCPU, 2, this);
    doubleValidator->setNotation(QDoubleValidator::StandardNotation);
    ui->lineEdit_cpu_core->setValidator(doubleValidator);
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

void CPUConfPage::getCPUInfo(container::HostConfig* cfg)
{
    if (cfg)
    {
        auto resourceCfg = cfg->mutable_resource_config();
        KLOG_INFO() << "cpu core:" << ui->lineEdit_cpu_core->text().toInt() * 1e9;
        resourceCfg->set_nano_cpus(ui->lineEdit_cpu_core->text().toInt() * 1e9);

        //调度优先级
        resourceCfg->set_cpu_shares(ui->lineEdit_sche_pri->text().toInt());
    }
}
