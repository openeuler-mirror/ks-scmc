#ifndef CONTAINERLIST_H
#define CONTAINERLIST_H

#include <QMenu>
#include <QStandardItemModel>
#include <QWidget>
#include "common/common-page.h"
#include "common/info-worker.h"
enum OPERATION_BUTTOM
{
    OPERATION_BUTTOM_RUN,
    OPERATION_BUTTOM_STOP,
    OPERATION_BUTTOM_RESTART,
    OPERATION_BUTTOM_DELETE,
    OPERATION_BUTTOM_MORE
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
    void onBtnRun(QModelIndex index);
    void onBtnStop();
    void onBtnStop(QModelIndex index);
    void onBtnRestart();
    void onBtnRestart(QModelIndex index);
    void onBtnDelete();
    void onActCopyConfig();
    void onActBatchUpdate();
    void onActBatchEdit();
    void onActBackup();

    void onMonitor(int row);
    void onEdit(int row);
    void onTerminal(int row);

    void getNodeListResult(const QPair<grpc::Status, node::ListReply> &);
    void getContainerListResult(const QPair<grpc::Status, container::ListReply> &);
    void getContainerStartResult(const QPair<grpc::Status, container::StartReply> &);
    void getContainerStopResult(const QPair<grpc::Status, container::StopReply> &);
    void getContainerRestartResult(const QPair<grpc::Status, container::RestartReply> &);
    void getContainerRemoveResult(const QPair<grpc::Status, container::RemoveReply> &);

private:
    void initButtons();
    void initTable();
    void initConnect();
    void getContainerList();
    void getContainerInspect(QMap<QString, QVariant> itemData);
    void getCheckedItemsId(std::map<int64_t, std::vector<std::string> > &ids);
    void getItemId(int row, std::map<int64_t, std::vector<std::string> > &ids);

private:
    QMenu *m_createMenu;
    QMenu *m_moreMenu;
    QMap<int, QPushButton *> m_opBtnMap;
    ContainerSetting *m_createCTSetting;
    ContainerSetting *m_editCTSetting;
    std::vector<int64_t> m_vecNodeId;
    QTimer *m_timer;
};

#endif  // CONTAINERLIST_H
