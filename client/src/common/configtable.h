#ifndef CONFIGTABLE_H
#define CONFIGTABLE_H

#include <QSharedPointer>
#include <QWidget>
#include "configmodel.h"
#include "configview.h"

namespace Ui
{
class ConfigTable;
}

class ConfigTable : public QWidget
{
    Q_OBJECT

public:
    explicit ConfigTable(ConfigTableType whichTable, QWidget *parent = nullptr);
    ~ConfigTable();
    void initwindow();
    void initTable();
    void paintEditor(int curRow);
    QList<QSharedPointer<ModelItem>> getAllData();

private slots:
    void saveRowSlot(int row);
    void editRowSlot(int row);
    void addRowSlot(int row);
    void removeRowSlot(int row);

private:
    Ui::ConfigTable *ui = nullptr;

    QSharedPointer<ConfigView> m_pHeaderView = nullptr;
    QSharedPointer<ConfigModel> m_pModel = nullptr;
    QSharedPointer<ConfigDelegate> m_pDelegate = nullptr;

    ConfigTableType m_ChooseTable;  //0:环境变量; 1:共享目录
};

#endif  // CONFIGTABLE_H
