#ifndef MESSAGEDIALOG_H
#define MESSAGEDIALOG_H

#include <QDialog>
#include <QDialogButtonBox>
#include <QMap>
#include <QMessageBox>

using ButtonRole = QDialogButtonBox::ButtonRole;
namespace Ui
{
class MessageDialog;
}

class MessageDialog : public QDialog
{
    Q_OBJECT

public:
    enum StandardButton
    {
        NoButton = 0x00000000,
        Ok = 0x00000400,
        Save = 0x00000800,
        SaveAll = 0x00001000,
        Open = 0x00002000,
        Yes = 0x00004000,
        YesToAll = 0x00008000,
        No = 0x00010000,
        NoToAll = 0x00020000,
        Abort = 0x00040000,
        Retry = 0x00080000,
        Ignore = 0x00100000,
        Close = 0x00200000,
        Cancel = 0x00400000,
        Discard = 0x00800000,
        Help = 0x01000000,
        Apply = 0x02000000,
        Reset = 0x04000000,
        RestoreDefaults = 0x08000000
    };
    Q_ENUM(StandardButton)
    Q_DECLARE_FLAGS(StandardButtons, StandardButton)
    Q_FLAG(StandardButtons)

public:
    explicit MessageDialog(QWidget *parent = 0);
    ~MessageDialog();
    static StandardButton message(const QString &title,
                                  const QString &summary,
                                  const QString &body,
                                  const QString icon,
                                  StandardButtons buttons);

    void setButtonSize(const QSize &size);
    void setDialogSize(const QSize &size);

private:
    void initUI();
    void addButton(StandardButton standButton);
    QAbstractButton *clickedButton();
    StandardButton standardButton(QAbstractButton *button);
    void setSummary(const QString &summary);
    void setBody(const QString &body);
    void setIcon(const QString icon);
    void setStandardButtons(StandardButtons buttons);

private slots:
    void onButtonClicked(QAbstractButton *button);

private:
    Ui::MessageDialog *ui;
    QDialogButtonBox *m_btnBox;
    QAbstractButton *m_clickedButton;
    QMap<QAbstractButton *, StandardButton> m_standBtnMap;
};
Q_DECLARE_OPERATORS_FOR_FLAGS(MessageDialog::StandardButtons)

#endif  // MESSAGEDIALOG_H
