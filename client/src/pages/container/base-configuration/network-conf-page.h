#ifndef NETWORKCONFPAGE_H
#define NETWORKCONFPAGE_H

#include <QWidget>
#include "common/info-worker.h"
namespace Ui
{
class NetworkConfPage;
}

class NetworkConfPage : public QWidget
{
    Q_OBJECT

public:
    explicit NetworkConfPage(QWidget *parent = nullptr);
    ~NetworkConfPage();
    void getNetworkInfo(container::CreateRequest *req);
    void updateNetworkInfo(int64_t node_id);

private:
    void getNetworkListResult(const QPair<grpc::Status, network::ListReply> &reply);

private:
    void initUI();
    void getNetworkInfo();

private:
    Ui::NetworkConfPage *ui;
};

#endif  // NETWORKCONFPAGE_H
