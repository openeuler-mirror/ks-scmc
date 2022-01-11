#include "envs-conf-page.h"
#include <QVBoxLayout>
#include "common/configtable.h"
#include "ui_envs-conf-page.h"
EnvsConfPage::EnvsConfPage(QWidget *parent) : QWidget(parent),
                                              ui(new Ui::EnvsConfPage)
{
    ui->setupUi(this);
    initUI();
}

EnvsConfPage::~EnvsConfPage()
{
    delete ui;
}

void EnvsConfPage::initUI()
{
    ConfigTable *configTable = new ConfigTable(CONFIG_TABLE_TYPE_ENV, this);
    QVBoxLayout *vLayout = new QVBoxLayout(this);
    vLayout->setMargin(20);
    vLayout->addWidget(configTable);
}
