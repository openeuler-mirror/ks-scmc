#ifndef CPUCONFPAGE_H
#define CPUCONFPAGE_H

#include <QWidget>
#include "common/def.h"
#include "common/info-worker.h"
namespace Ui
{
class CPUConfPage;
}

class CPUConfPage : public QWidget
{
    Q_OBJECT

public:
    explicit CPUConfPage(double totalCPU, QWidget *parent = nullptr);
    ~CPUConfPage();
    void setCPUInfo(container::HostConfig *cfg);
    void getCPUInfo(container::ResourceConfig *cfg);

private:
    Ui::CPUConfPage *ui;
    double m_totalCPU;
};

#endif  // CPUCONFPAGE_H
