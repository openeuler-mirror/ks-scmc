#ifndef MESSAGEDIALOG_H
#define MESSAGEDIALOG_H

#include <QDialog>

namespace Ui
{
class MessageDialog;
}

class MessageDialog : public QDialog
{
    Q_OBJECT

public:
    explicit MessageDialog(QWidget *sender, QWidget *parent = nullptr);
    ~MessageDialog();
    void setTitle(QString title);
    void setSummary(QString summary);
    void setBody(QString body);
    void setIcon(QString icon);
    void setWidth(int width);

protected:
    void paintEvent(QPaintEvent *event) override;

private slots:
    void onConfirm();
    void onCancel();

signals:
    void sigConfirm(QWidget *sender);
    void sigCancel();

private:
    void initUI();

private:
    Ui::MessageDialog *ui;
    QWidget *m_sender;
};

#endif  // MESSAGEDIALOG_H
