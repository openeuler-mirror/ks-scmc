#include "high-availability-page.h"
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

    connect(ui->cb_high_avail_policy, QOverload<const QString &>::of(&QComboBox::activated), this, &HighAvailabilityPage::onCbActivated);
}
