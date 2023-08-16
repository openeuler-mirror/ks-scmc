#ifndef NODELIST_H
#define NODELIST_H

#include <QWidget>
#include "common/common-page.h"

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

private:
    void initButtons();
    void initTable();

private:
    NodeAddition *m_nodeAddition;
};

#endif  // NODELIST_H
