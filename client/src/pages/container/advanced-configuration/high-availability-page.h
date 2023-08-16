#ifndef HIGHAVAILABILITYPAGE_H
#define HIGHAVAILABILITYPAGE_H

#include <QWidget>

namespace Ui
{
class HighAvailabilityPage;
}

class HighAvailabilityPage : public QWidget
{
    Q_OBJECT

public:
    explicit HighAvailabilityPage(QWidget *parent = nullptr);
    ~HighAvailabilityPage();
    void setHighAvailInfo();
    //getHighAvailInf();

private slots:
    void onCbActivated(QString text);
    void setLineEditVisible(bool visible);

private:
    void initUI();
    bool m_isVisible;

private:
    Ui::HighAvailabilityPage *ui;
};

#endif  // HIGHAVAILABILITYPAGE_H
