#include "memory-conf-page.h"
#include <kiran-log/qt5-log-i.h>
#include "ui_memory-conf-page.h"
MemoryConfPage::MemoryConfPage(QWidget *parent) : QWidget(parent),
                                                  ui(new Ui::MemoryConfPage)
{
    ui->setupUi(this);
    QList<QComboBox *> comboboxs = this->findChildren<QComboBox *>();
    foreach (QComboBox *cb, comboboxs)
    {
        cb->addItems(QStringList() << "MB"
                                   << "GB");
    }
    QRegExp regExp("[0-9]+.?[0-9]*");
    ui->lineEdit_soft_limit->setValidator(new QRegExpValidator(regExp));
    ui->lineEdit_max_limit->setValidator(new QRegExpValidator(regExp));
}

MemoryConfPage::~MemoryConfPage()
{
    delete ui;
}

void MemoryConfPage::setMemoryInfo(container::HostConfig *cfg)
{
    if (cfg)
    {
        auto resourceCfg = cfg->mutable_resource_config();
        int memLimit = limitDataHandle(true, resourceCfg->mem_limit(), ui->cb_max_unit);
        int sofgLimit = limitDataHandle(true, resourceCfg->mem_soft_limit(), ui->cb_soft_unit);
        KLOG_INFO() << resourceCfg->mem_limit() << resourceCfg->mem_soft_limit();

        ui->lineEdit_soft_limit->setText(QString("%1").arg(sofgLimit));
        ui->lineEdit_max_limit->setText(QString("%1").arg(memLimit));
    }
}

ErrorCode MemoryConfPage::getMemoryInfo(container::ResourceConfig *cfg)
{
    if (cfg)
    {
        auto softLimit = limitDataHandle(false, ui->lineEdit_soft_limit->text().toInt(), ui->cb_soft_unit);
        KLOG_INFO() << "Memory soft limit: " << softLimit;

        auto maxLimit = limitDataHandle(false, ui->lineEdit_max_limit->text().toInt(), ui->cb_max_unit);
        KLOG_INFO() << "Memory max limit: " << maxLimit;

        if (softLimit <= maxLimit)
        {
            cfg->set_mem_limit(maxLimit);
            cfg->set_mem_soft_limit(softLimit);
            return NO_ERROR;
        }
        else
            return INPUT_ARG_ERROR;
    }
    return CONFIG_ARG_ERROR;
}

int MemoryConfPage::limitDataHandle(bool stom, int originData, QComboBox *unitWidget)
{
    QString unit = unitWidget->currentText();
    int limit;
    if (stom)  //bites->M/GB
    {
        if (unit == "MB")
            limit = originData >> 20;
        else
            limit = originData >> 30;
    }
    else  //M/GB->bites
    {
        if (unit == "MB")
            limit = originData << 20;
        else
            limit = originData << 30;
    }
    return limit;
}
