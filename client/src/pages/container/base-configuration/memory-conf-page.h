#ifndef MEMORYCONFPAGE_H
#define MEMORYCONFPAGE_H

#include <QWidget>
#include "common/def.h"
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
    void setMemoryInfo(MemoryInfo memoryInfo);
    MemoryInfo getMemoryInfo();

private:
    Ui::MemoryConfPage *ui;
};

#endif  // MEMORYCONFPAGE_H
