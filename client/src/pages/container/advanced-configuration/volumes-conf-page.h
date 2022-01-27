#ifndef VOLUMESCONFPAGE_H
#define VOLUMESCONFPAGE_H

#include <QWidget>
#include "common/configtable.h"
#include "common/info-worker.h"
namespace Ui
{
class VolumesConfPage;
}

class VolumesConfPage : public QWidget
{
    Q_OBJECT

public:
    explicit VolumesConfPage(QWidget *parent = nullptr);
    ~VolumesConfPage();
    void getVolumeInfo(container::HostConfig *cfg);

private:
    void initUI();

private:
    Ui::VolumesConfPage *ui;
    ConfigTable *m_configTable;
};

#endif  // VOLUMESCONFPAGE_H
