#include "network-conf-page.h"
#include <QRegExpValidator>
#include "ui_network-conf-page.h"
NetworkConfPage::NetworkConfPage(QWidget *parent) : QWidget(parent),
                                                    ui(new Ui::NetworkConfPage)
{
    ui->setupUi(this);
    initUI();
}

NetworkConfPage::~NetworkConfPage()
{
    delete ui;
}

void NetworkConfPage::initUI()
{
    ui->lineEdit_ip->setPlaceholderText(tr("Default auto-assignment when not config"));
    ui->lineEdit_ip->setToolTip(tr("Available network segment: 192.168.1.1~20"));
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

    //    QRegExp rxMac("/b(?:[0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}/b");
    //    ui->lineEdit_mac->setValidator(new QRegExpValidator(rxMac));
    ui->lineEdit_mac->setPlaceholderText(tr("Default auto-assignment when not config"));
}

QStringList NetworkConfPage::getVirtualNetCard()
{
}
