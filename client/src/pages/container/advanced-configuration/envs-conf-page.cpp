#include "envs-conf-page.h"
#include <QVBoxLayout>
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

void EnvsConfPage::getEnvInfo(container::ContainerConfig *cfg)
{
    if (cfg)
    {
        auto env = cfg->mutable_env();
        auto itemList = m_configTable->getAllData();
        for (auto item : itemList)
        {
            auto key = item->m_firstColVal;
            auto value = item->m_secondColVal;
            env->insert({key.toStdString(), value.toStdString()});
        }
    }
}

void EnvsConfPage::initUI()
{
    m_configTable = new ConfigTable(CONFIG_TABLE_TYPE_ENV, this);
    QVBoxLayout *vLayout = new QVBoxLayout(this);
    vLayout->setMargin(20);
    vLayout->addWidget(m_configTable);
}
