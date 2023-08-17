#ifndef NODELIST_H
#define NODELIST_H

#include <QWidget>
#include "common/common-page.h"
#include "common/info-worker.h"

struct nodeInfo_s
{
    nodeInfo_s(std::string _name, std::string _addr, std::string _comment) : name(_name), address(_addr), comment(_comment){}
    std::string name;
    std::string address;
    std::string comment;
};

class NodeAddition;
class NodeList : public CommonPage
{
    Q_OBJECT
public:
    explicit NodeList(QWidget *parent = nullptr);
    ~NodeList();
    void updateInfo(QString keyword = "");  //刷新表格

private slots:
    void onCreateNode();
    void onRemoveNode();
    void onMonitor(int row);
    void onSaveSlot(QMap<QString, QString> Info);
    void getListResult(const QPair<grpc::Status, node::ListReply> &);
    void getCreateResult(const QPair<grpc::Status, node::CreateReply> &);
    void getRemoveResult(const QPair<grpc::Status, node::RemoveReply> &);

private:
    void initButtons();
    void initTable();
    void initNodeConnect();
    void getNodeList();

private:
    NodeAddition *m_nodeAddition;
    std::map <int64_t, std::string> m_mapStatus;
};

#endif  // NODELIST_H
