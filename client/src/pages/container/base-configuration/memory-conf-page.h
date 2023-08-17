#ifndef MEMORYCONFPAGE_H
#define MEMORYCONFPAGE_H

#include <QComboBox>
#include <QLineEdit>
#include <QWidget>
#include "common/def.h"
#include "common/info-worker.h"
namespace Ui
{
class MemoryConfPage;
}

class MemoryConfPage : public QWidget
{
    Q_OBJECT

public:
    explicit MemoryConfPage(QWidget *parent = nullptr);
    ~MemoryConfPage();
    void setMemoryInfo(container::HostConfig *cfg);
    ErrorCode getMemoryInfo(container::ResourceConfig *cfg);

private:
    //bites->M/GB,M/GB->bites
    //stom: Small to Max
    int limitDataHandle(bool stom, int originData, QComboBox *unitWidget);

private:
    Ui::MemoryConfPage *ui;
};

#endif  // MEMORYCONFPAGE_H
