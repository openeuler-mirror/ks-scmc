#ifndef OPERATEWIDGET_H
#define OPERATEWIDGET_H

#include <QWidget>

namespace Ui {
class OperateWidget;
}

class OperateWidget : public QWidget
{
    Q_OBJECT

public:
    explicit OperateWidget(int curRow, int page, QWidget *parent = nullptr);
    ~OperateWidget();
    int getCurRow();
    int getCurPage();
    void setPage(int page);

private:
    void initControl();
    void initConnections();

signals:
    void sendSaveSig(OperateWidget*);
    void sendEditSig(OperateWidget*);
    void sendAddSig(OperateWidget*);
    void sendDeleteSig(OperateWidget*);

private slots:
    void sendSave();
    void sendEdit();
    void sendAdd();
    void sendDelete();

private:
    Ui::OperateWidget *ui;
    int m_CurRow;
};

#endif // OPERATEWIDGET_H
