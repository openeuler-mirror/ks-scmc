#include "node-addition.h"
#include <QRegExpValidator>
#include "common/message-dialog.h"
#include "ui_node-addition.h"
NodeAddition::NodeAddition(QWidget *parent) : QWidget(parent),
                                              ui(new Ui::NodeAddition)
{
    ui->setupUi(this);
    setAttribute(Qt::WA_DeleteOnClose);
    setWindowIcon(QIcon(":/images/logo.png"));
    setWindowTitle(tr("Add Node"));

    QRegExp rx("\\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\b");
    ui->lineEdit_node_ip->setValidator(new QRegExpValidator(rx, parent));

    connect(ui->btn_save, &QPushButton::clicked, this, &NodeAddition::onSave);
    connect(ui->btn_cancel, &QPushButton::clicked, this, &NodeAddition::onCancel);
}

NodeAddition::~NodeAddition()
{
    delete ui;
}

void NodeAddition::onSave()
{
    QString name = ui->lineEdit_node_name->text();
    QString ip = ui->lineEdit_node_ip->text();
    if (name.isEmpty() || ip.isEmpty())
    {
        MessageDialog::message(tr("Add node"),
                               tr("Add node failed!"),
                               tr("Please improve the content!"),
                               ":/images/warning.svg",
                               MessageDialog::StandardButton::Ok);
        return;
    }
    QMap<QString, QString> newNodeInfo;
    newNodeInfo.insert("Node Name", name);
    newNodeInfo.insert("Node IP", ip);
    emit sigSave(newNodeInfo);
    this->close();
}

void NodeAddition::onCancel()
{
    this->close();
}
