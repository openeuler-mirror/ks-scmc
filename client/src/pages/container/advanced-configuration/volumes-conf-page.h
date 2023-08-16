#ifndef VOLUMESCONFPAGE_H
#define VOLUMESCONFPAGE_H

#include <QWidget>

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

private:
    void initUI();

private:
    Ui::VolumesConfPage *ui;
};

#endif  // VOLUMESCONFPAGE_H
