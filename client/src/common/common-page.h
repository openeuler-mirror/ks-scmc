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

class MaskWidget;
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
    void setOpBtnEnabled(bool enabled);
    void setTableColNum(int num);
    void setTableRowNum(int num);
    void setTableItem(int row, int col, QStandardItem *item);
    void setTableItems(int row, int col, QList<QStandardItem *> items);
    void setTableActions(int col, QStringList actionIcons);
    void setSortableCol(QList<int> cols);
    void setHeaderSections(QStringList names);
    void setTableDefaultContent(QList<int> actionCol, QString text);
    void clearText();
    int getTableRowCount();
    QStandardItem *getItem(int row, int col);
    QList<QMap<QString, QVariant>> getCheckedItemInfo(int col);
    void sleep(int sec);

private:
    void initUI();
    void adjustTableSize();

protected:
    bool eventFilter(QObject *watched, QEvent *event);
    virtual void resizeEvent(QResizeEvent *event) override;

signals:
    void sigMonitor(int row);
    void sigEdit(int row);
    void sigTerminal(int row);
    void sigRun(QModelIndex index);
    void sigStop(QModelIndex index);
    void sigRestart(QModelIndex index);
    void sigTableHeightChanged(int height);

private slots:
    void onMonitor(int row);
    void onTerminal(int row);
    void onEdit(int row);
    void onActRun(QModelIndex index);
    void onActStop(QModelIndex index);
    void onActRestart(QModelIndex index);
    void onRefreshTimeout();
    void search();
    void refresh();
    void onHeaderCkbTog(bool toggled);

private:
    Ui::CommonPage *ui;
    QString m_keyword;
    QStandardItemModel *m_model;
    HeaderView *m_headerView;
    QTimer *m_searchTimer;
    QTimer *m_refreshBtnTimer;
    int m_actionCol;
    MaskWidget *m_maskWidget;
};

#endif  // COMMONPAGE_H
