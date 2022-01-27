#include "high-availability-page.h"
#include <kiran-log/qt5-log-i.h>
#include "ui_high-availability-page.h"
HighAvailabilityPage::HighAvailabilityPage(QWidget *parent) : QWidget(parent),
                                                              ui(new Ui::HighAvailabilityPage)
{
    ui->setupUi(this);
    initUI();
}

HighAvailabilityPage::~HighAvailabilityPage()
{
    delete ui;
}

void HighAvailabilityPage::getRestartPolicy(container::HostConfig *cfg)
{
    if (cfg)
    {
        auto policy = cfg->mutable_restart_policy();

        KLOG_INFO() << "Policy :" << ui->cb_high_avail_policy->currentText() << "times: " << ui->lineEdit_times->text();
        policy->set_name(ui->cb_high_avail_policy->currentText().toStdString());
        if (ui->lineEdit_times->isVisible())
            policy->set_max_retry(ui->lineEdit_times->text().toInt());
    }
}

void HighAvailabilityPage::onCbActivated(QString text)
{
    if (text == "on-failure")
    {
        if (m_isVisible == false)
            setLineEditVisible(true);
    }
    else
    {
        if (m_isVisible == true)
            setLineEditVisible(false);
    }
}

void HighAvailabilityPage::setLineEditVisible(bool visible)
{
    m_isVisible = visible;
    ui->lab_auto_pull_time->setVisible(visible);
    ui->lineEdit_times->setVisible(visible);
    ui->lab_time->setVisible(visible);
}

void HighAvailabilityPage::initUI()
{
    setLineEditVisible(false);
    ui->cb_high_avail_policy->addItems(QStringList()
                                       << tr("disabled")
                                       << tr("always")
                                       << tr("on-failure")
                                       << tr("unless-stopped"));

    ui->lineEdit_times->setValidator(new QIntValidator(0, 20, this));

    connect(ui->cb_high_avail_policy, QOverload<const QString &>::of(&QComboBox::activated), this, &HighAvailabilityPage::onCbActivated);
}
