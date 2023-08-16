#ifndef CONTAINERLIST_H
#define CONTAINERLIST_H

#include <QMenu>
#include <QStandardItemModel>
#include <QWidget>
#include "common/common-page.h"

enum OPERATION_BUTTOM
{
    OPERATION_BUTTOM_RUN,
    OPERATION_BUTTOM_STOP,
    OPERATION_BUTTOM_RESTART,
    OPERATION_BUTTOM_DELETE,
    OPERATION_BUTTOM_MORE
};

enum ContainerSettingType
{
    CONTAINER_SETTING_TYPE_CREATE,
    CONTAINER_SETTING_TYPE_EDIT
};

class ContainerSetting;
class ContainerList : public CommonPage
{
    Q_OBJECT

public:
    explicit ContainerList(QWidget *parent = nullptr);
    ~ContainerList();
    void updateInfo(QString keyword = "");  //刷新表格

private slots:
    void onBtnCreate();
    void onBtnRun();
    void onBtnStop();
    void onBtnRestart();
    void onBtnDelete();
    void onActCopyConfig();
    void onActBatchUpdate();
    void onActBatchEdit();
    void onActBackup();

    void onMonitor(int row);
    void onEdit(int row);
    void onTerminal(int row);

private:
    void initButtons();
    void initTable();
    void initContianerSetting(ContainerSetting *window, ContainerSettingType type);
    void insertContainerInfo();

private:
    QMenu *m_createMenu;
    QMenu *m_moreMenu;
    QStandardItemModel *m_model;
    QMap<int, QPushButton *> m_opBtnMap;
    ContainerSetting *m_createCTSetting;
    ContainerSetting *m_editCTSetting;
};

#endif  // CONTAINERLIST_H
