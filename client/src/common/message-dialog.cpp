#include "message-dialog.h"
#include <QPainter>
#include <QStyleOption>
#include "ui_message-dialog.h"
MessageDialog::MessageDialog(QWidget *sender, QWidget *parent) : QDialog(parent),
                                                                 ui(new Ui::MessageDialog),
                                                                 m_sender(sender)
{
    ui->setupUi(this);
    connect(ui->btn_cancel, &QToolButton::clicked, this, &MessageDialog::onCancel);
    connect(ui->btn_confirm, &QToolButton::clicked, this, &MessageDialog::onConfirm);
}

MessageDialog::~MessageDialog()
{
    delete ui;
}

void MessageDialog::setTitle(QString title)
{
    this->setWindowTitle(title);
}

void MessageDialog::setSummary(QString summary)
{
    ui->lab_summary->setText(summary);
}

void MessageDialog::setBody(QString body)
{
    ui->lab_body->setText(body);
}

void MessageDialog::setIcon(QString icon)
{
    ui->lab_icon->setStyleSheet(QString("#lab_icon{image:url(%1)}").arg(icon));
}

void MessageDialog::setWidth(int width)
{
    setFixedWidth(width);
}

void MessageDialog::paintEvent(QPaintEvent *event)
{
    QStyleOption opt;
    opt.init(this);
    QPainter p(this);
    style()->drawPrimitive(QStyle::PE_Widget, &opt, &p, this);
}

void MessageDialog::onConfirm()
{
    emit sigConfirm(m_sender);
    close();
}

void MessageDialog::onCancel()
{
    close();
}
