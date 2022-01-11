#ifndef NETWORKCONFPAGE_H
#define NETWORKCONFPAGE_H

#include <QWidget>

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

private:
    void initUI();
    QStringList getVirtualNetCard();

private:
    Ui::NetworkConfPage *ui;
};

#endif  // NETWORKCONFPAGE_H
