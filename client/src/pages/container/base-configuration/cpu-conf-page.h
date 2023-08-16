#ifndef CPUCONFPAGE_H
#define CPUCONFPAGE_H

#include <QWidget>
#include "common/def.h"
namespace Ui
{
class CPUConfPage;
}

class CPUConfPage : public QWidget
{
    Q_OBJECT

public:
    explicit CPUConfPage(QWidget *parent = nullptr);
    ~CPUConfPage();
    void setCPUInfo(CPUInfo cpuInfo);
    CPUInfo getCPUInfo();

private:
    Ui::CPUConfPage *ui;
};

#endif  // CPUCONFPAGE_H
