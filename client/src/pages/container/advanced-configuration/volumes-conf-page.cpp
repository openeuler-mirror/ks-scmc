#include "volumes-conf-page.h"
#include <QVBoxLayout>
#include "common/configtable.h"
#include "ui_volumes-conf-page.h"

VolumesConfPage::VolumesConfPage(QWidget *parent) : QWidget(parent),
                                                    ui(new Ui::VolumesConfPage)
{
    ui->setupUi(this);
    initUI();
}

VolumesConfPage::~VolumesConfPage()
{
    delete ui;
}

void VolumesConfPage::initUI()
{
    ConfigTable *configTable = new ConfigTable(CONFIG_TABLE_TYPE_VOLUMES, this);
    QVBoxLayout *vLayout = new QVBoxLayout(this);
    vLayout->setMargin(20);
    vLayout->addWidget(configTable);
}
