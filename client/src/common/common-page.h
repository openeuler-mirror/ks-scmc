#ifndef COMMONPAGE_H
#define COMMONPAGE_H

#include <QPushButton>
#include <QStandardItemModel>
#include <QToolButton>
#include <QWidget>
namespace Ui
{
class CommonPage;
}

class HeaderView;
class CommonPage : public QWidget
{
    Q_OBJECT

public:
    explicit CommonPage(QWidget *parent = 0);
    virtual ~CommonPage();
    virtual void updateInfo(QString keyword = "") = 0;
    void setBusy(bool status);
    void clearTable();
    void addOperationButton(QToolButton *);
    void addOperationButtons(QList<QPushButton *>);
    void setTableColNum(int num);
    void setTableRowNum(int num);
    void setTableItem(int col, int row, QStandardItem *item);
    void setTableItems(int row, QList<QStandardItem *> items);
    void setTableActions(int col, QStringList actionIcons);
    void setSortableCol(QList<int> cols);
    void setHeaderSections(QStringList names);
    QList<QMap<QString, QVariant>> getCheckedRowInfo();

private:
    void initUI();

signals:
    void sigMonitor(int row);
    void sigEdit(int row);
    void sigTerminal(int row);
    void sigRun(QModelIndex index);
    void sigStop(QModelIndex index);
    void sigRestart(QModelIndex index);

private slots:
    void onMonitor(int row);
    void onTerminal(int row);
    void onEdit(int row);
    void onActRun(QModelIndex index);
    void onActStop(QModelIndex index);
    void onActRestart(QModelIndex index);
    void search();
    void onHeaderCkbTog(bool toggled);

private:
    Ui::CommonPage *ui;
    QString m_keyword;
    QStandardItemModel *m_model;
    HeaderView *m_headerView;
    QList<QMap<QString, QVariant>> m_checkedRowInfo;  //containerId,nodeId
};

#endif  // COMMONPAGE_H
