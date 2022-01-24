#ifndef HIGHAVAILABILITYPAGE_H
#define HIGHAVAILABILITYPAGE_H

#include <QWidget>
#include "common/info-worker.h"
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
    void setRestartPolicy(container::HostConfig *cfg);
    void getRestartPolicy(container::RestartPolicy *cfg);

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
