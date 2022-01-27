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

void MemoryConfPage::setMemoryInfo(MemoryInfo memoryInfo)
{
    ui->lineEdit_soft_limit->setText(QString("%1").arg(memoryInfo.softLimit));
    ui->lineEdit_max_limit->setText(QString("%1").arg(memoryInfo.maxLimit));
}

void MemoryConfPage::getMemoryInfo(container::HostConfig *cfg)
{
    auto resourceCfg = cfg->mutable_resource_config();

    auto softLimit = getLimitData(ui->lineEdit_soft_limit, ui->cb_soft_unit);
    KLOG_INFO() << "Memory soft limit: " << softLimit;

    auto maxLimit = getLimitData(ui->lineEdit_max_limit, ui->cb_max_unit);
    KLOG_INFO() << "Memory max limit: " << maxLimit;

    if (softLimit <= maxLimit)
    {
        resourceCfg->set_mem_limit(maxLimit);
        resourceCfg->set_mem_soft_limit(softLimit);
    }
}

qlonglong MemoryConfPage::getLimitData(QLineEdit *inputWidget, QComboBox *unitWidget)
{
    QString unit = unitWidget->currentText();
    qlonglong limit;
    if (unit == "MB")
        limit = inputWidget->text().toLongLong() << 20;
    else
        limit = inputWidget->text().toLongLong() << 30;
    return limit;
}
