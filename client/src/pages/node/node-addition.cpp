#include "node-addition.h"
#include <QRegExpValidator>
#include "ui_node-addition.h"
NodeAddition::NodeAddition(QWidget *parent) : QWidget(parent),
                                              ui(new Ui::NodeAddition)
{
    ui->setupUi(this);
    setAttribute(Qt::WA_DeleteOnClose);
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
    QMap<QString, QString> newNodeInfo;
    newNodeInfo.insert("Node Name", ui->lineEdit_node_name->text());
    newNodeInfo.insert("Node IP", ui->lineEdit_node_ip->text());
    emit sigSave(newNodeInfo);
    this->close();
}

void NodeAddition::onCancel()
{
    this->close();
}
