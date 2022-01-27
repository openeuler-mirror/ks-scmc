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
    void setMemoryInfo(MemoryInfo memoryInfo);
    void getMemoryInfo(container::HostConfig *cfg);

private:
    qlonglong getLimitData(QLineEdit *inputWidget, QComboBox *unitWidget);

private:
    Ui::MemoryConfPage *ui;
};

#endif  // MEMORYCONFPAGE_H
