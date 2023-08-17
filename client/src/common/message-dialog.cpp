#include "message-dialog.h"
#include <kiran-log/qt5-log-i.h>
#include <QAbstractButton>
#include <QDebug>
#include <QMetaEnum>
#include <QPushButton>
#include "ui_message-dialog.h"
static QMap<MessageDialog::StandardButton, QPair<QString, ButtonRole>> standardButtonInfoMap;

MessageDialog::MessageDialog(QWidget *parent) : QDialog(parent),
                                                ui(new Ui::MessageDialog)
{
    ui->setupUi(this);
    initUI();

    connect(m_btnBox, &QDialogButtonBox::clicked, this, &MessageDialog::onButtonClicked);
}

MessageDialog::~MessageDialog()
{
    delete ui;
}

void MessageDialog::setSummary(const QString &summary)
{
    ui->label_summary->setText(summary);
}

void MessageDialog::setBody(const QString &body)
{
    ui->label_body->setText(body);
}

void MessageDialog::setIcon(const QString icon)
{
    QPixmap pixmap(icon);
    ui->lab_icon->setPixmap(pixmap);
}

void MessageDialog::setStandardButtons(StandardButtons buttons)
{
    for (auto iter = m_standBtnMap.begin(); iter != m_standBtnMap.end(); iter++)
    {
        QAbstractButton *button = iter.key();
        m_btnBox->removeButton(button);
        delete button;
    }
    m_standBtnMap.clear();

    QMetaEnum metaEnum = QMetaEnum::fromType<StandardButton>();
    for (int i = 0; i < metaEnum.keyCount(); i++)
    {
        if (buttons & metaEnum.value(i))
        {
            addButton(static_cast<StandardButton>(metaEnum.value(i)));
        }
    }
}

MessageDialog::StandardButton MessageDialog::message(const QString &title, const QString &summary, const QString &body, const QString icon, StandardButtons buttons)
{
    MessageDialog msgDlg;
    msgDlg.setWindowTitle(title);
    msgDlg.setSummary(summary);
    msgDlg.setBody(body);
    msgDlg.setIcon(icon);
    msgDlg.setStandardButtons(buttons);
    if (msgDlg.exec() == -1)
        return StandardButton::Cancel;
    return msgDlg.standardButton(msgDlg.clickedButton());
}

void MessageDialog::setButtonSize(const QSize &size)
{
    QList<QAbstractButton *> btns = m_btnBox->buttons();
    for (QAbstractButton *btn : btns)
    {
        btn->setFixedSize(size);
    }
}

void MessageDialog::setDialogSize(const QSize &size)
{
    setFixedSize(size);
}

void MessageDialog::initUI()
{
    setWindowFlags(Qt::WindowCloseButtonHint);
    setWindowIcon(QIcon(":/images/logo.png"));
    if (standardButtonInfoMap.isEmpty())
    {
        standardButtonInfoMap = {
            {StandardButton::Ok, {QObject::tr("OK"), ButtonRole::AcceptRole}},
            {StandardButton::Save, {QObject::tr("Save"), ButtonRole::AcceptRole}},
            {StandardButton::SaveAll, {QObject::tr("Save All"), ButtonRole::AcceptRole}},
            {StandardButton::Open, {QObject::tr("Open"), ButtonRole::AcceptRole}},
            {StandardButton::Yes, {QObject::tr("Yes"), ButtonRole::YesRole}},
            {StandardButton::YesToAll, {QObject::tr("Yes to All"), ButtonRole::YesRole}},
            {StandardButton::No, {QObject::tr("No"), ButtonRole::NoRole}},
            {StandardButton::NoToAll, {QObject::tr("No to All"), ButtonRole::NoRole}},
            {StandardButton::Abort, {QObject::tr("Abort"), ButtonRole::RejectRole}},
            {StandardButton::Retry, {QObject::tr("Retry"), ButtonRole::AcceptRole}},
            {StandardButton::Ignore, {QObject::tr("Ignore"), ButtonRole::AcceptRole}},
            {StandardButton::Close, {QObject::tr("Close"), ButtonRole::RejectRole}},
            {StandardButton::Cancel, {QObject::tr("Cancel"), ButtonRole::RejectRole}},
            {StandardButton::Discard, {QObject::tr("Discard"), ButtonRole::DestructiveRole}},
            {StandardButton::Help, {QObject::tr("Help"), ButtonRole::HelpRole}},
            {StandardButton::Apply, {QObject::tr("Apply"), ButtonRole::ApplyRole}},
            {StandardButton::Reset, {QObject::tr("Reset"), ButtonRole::ResetRole}},
            {StandardButton::RestoreDefaults, {QObject::tr("Restore Defaults"), ButtonRole::ResetRole}}};
    }

    m_btnBox = new QDialogButtonBox(this);
    ui->hlayout_btns->addWidget(m_btnBox);
    QLayout *layout = m_btnBox->layout();
    layout->setSpacing(20);
}

MessageDialog::StandardButton MessageDialog::standardButton(QAbstractButton *button)
{
    auto iter = m_standBtnMap.find(button);
    return iter.value();
}

void MessageDialog::onButtonClicked(QAbstractButton *button)
{
    m_clickedButton = button;
    done(0);
}

void MessageDialog::addButton(MessageDialog::StandardButton standButton)
{
    ///判断按钮类型正确
    auto buttonInfoIter = standardButtonInfoMap.find(standButton);
    if (buttonInfoIter == standardButtonInfoMap.end())
    {
        qWarning() << "wrong standard button type," << standButton;
        return;
    }
    KLOG_INFO() << buttonInfoIter.value().first;
    QPushButton *button = new QPushButton(this);
    button->setFixedSize(QSize(100, 40));
    button->setText(buttonInfoIter.value().first);

    ///生成ObjectName，eg:KiranStandardButton::Ok->btn_Ok
    QMetaEnum metaEnum = QMetaEnum::fromType<StandardButton>();
    QString standardButtonString = metaEnum.valueToKey(standButton);
    standardButtonString.replace(" ", "_");
    QString objectName = QString("btn_%1").arg(standardButtonString);
    button->setObjectName(objectName);

    m_standBtnMap.insert(qobject_cast<QAbstractButton *>(button), standButton);
    m_btnBox->addButton(button, buttonInfoIter.value().second);
}

QAbstractButton *MessageDialog::clickedButton()
{
    return m_clickedButton;
}
