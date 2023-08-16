#include "network-conf-page.h"
#include <kiran-log/qt5-log-i.h>
#include <QRegExpValidator>
#include "ui_network-conf-page.h"
NetworkConfPage::NetworkConfPage(QWidget *parent) : QWidget(parent),
                                                    ui(new Ui::NetworkConfPage)
{
    ui->setupUi(this);
    initUI();
    getNetworkInfo();
    connect(&InfoWorker::getInstance(), &InfoWorker::listNetworkFinished, this, &NetworkConfPage::getNetworkListResult);
}

NetworkConfPage::~NetworkConfPage()
{
    delete ui;
}

void NetworkConfPage::getNetworkInfo(container::CreateRequest *req)
{
    auto cfg = req->mutable_network_config();
    QString str = ui->cb_virt_networkcard->currentText();
    QString name = str.split(" ").first();  // 网卡名

    container::EndpointSetting setting;
    auto ipam = setting.mutable_ipam_config();
    ipam->set_ipv4_address(ui->lineEdit_ip->text().toStdString());
    setting.set_mac_address(ui->lineEdit_mac->text().toStdString());

    cfg->insert({name.toStdString(), setting});
}

void NetworkConfPage::updateNetworkInfo(int64_t node_id)
{
    InfoWorker::getInstance().listNetwork(node_id);
}

void NetworkConfPage::getNetworkListResult(const QPair<grpc::Status, network::ListReply> &reply)
{
    if (reply.first.ok())
    {
        ui->cb_virt_networkcard->clear();
        QString gatewaykStr;
        QString tips;
        for (auto bridge : reply.second.bridge_if())
        {
            auto name = bridge.name();
            auto gateway = bridge.gateway();
            KLOG_INFO() << QString::fromStdString(name) << QString::fromStdString(gateway);

            if (!gateway.empty())
            {
                gatewaykStr = QString("%1 (%2:%3)")
                                  .arg(QString::fromStdString(name))
                                  .arg(tr("Gateway"))
                                  .arg(QString::fromStdString(gateway));
                KLOG_INFO() << "gatewaykStr: " << gatewaykStr;
                ui->cb_virt_networkcard->addItem(gatewaykStr);
            }
            else
            {
                gatewaykStr = QString("%1 (%2:/)")
                                  .arg(QString::fromStdString(name))
                                  .arg(tr("Gateway"));
                ui->cb_virt_networkcard->addItem(gatewaykStr);
            }

            if (bridge.has_ip_range())
            {  // ip_range优先
                auto addr = bridge.ip_range().addr();
                auto prefix_len = bridge.ip_range().prefix_len();
                tips = QString("Available network segment: %1/%2").arg(QString::fromStdString(addr)).arg(prefix_len);
                ui->lineEdit_ip->setToolTip(tips);
            }
            else if (bridge.has_subnet())
            {  // 没有ip_range 使用subnet
                auto addr = bridge.subnet().addr();
                auto prefix_len = bridge.subnet().prefix_len();
                tips = QString("Available network segment: %1/%2").arg(QString::fromStdString(addr)).arg(prefix_len);
                ui->lineEdit_ip->setToolTip(tips);
            }
        }
    }
}

void NetworkConfPage::initUI()
{
    ui->lineEdit_ip->setPlaceholderText(tr("Default auto-assignment when not config"));
    ui->lineEdit_ip->setStyleSheet("QToolTip{"
                                   "background-color: rgb(255,255,255);"
                                   "color:#000000;"
                                   "border-radius: 6px;"
                                   "border:0px solid rgb(0,0,0);"
                                   "outline:none; "
                                   "min-height:30px;"
                                   "}");
    QRegExp rx("\\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\b");
    ui->lineEdit_ip->setValidator(new QRegExpValidator(rx));

    QRegExp rxMac("([0-9a-fA-F]{2})((:[0-9a-fA-F]{2}){5})");
    ui->lineEdit_mac->setValidator(new QRegExpValidator(rxMac));
    ui->lineEdit_mac->setPlaceholderText(tr("Default auto-assignment when not config"));
}

void NetworkConfPage::getNetworkInfo()
{
}
