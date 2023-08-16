#ifndef ENVSCONFPAGE_H
#define ENVSCONFPAGE_H

#include <QWidget>
#include "common/configtable.h"
#include "common/info-worker.h"
namespace Ui
{
class EnvsConfPage;
}

class EnvsConfPage : public QWidget
{
    Q_OBJECT

public:
    explicit EnvsConfPage(QWidget *parent = nullptr);
    ~EnvsConfPage();
    void getEnvInfo(container::ContainerConfig *cfg);

private:
    void initUI();

private:
    Ui::EnvsConfPage *ui;
    ConfigTable *m_configTable;
};

#endif  // ENVSCONFPAGE_H
