#ifndef ENVSCONFPAGE_H
#define ENVSCONFPAGE_H

#include <QWidget>

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

private:
    void initUI();

private:
    Ui::EnvsConfPage *ui;
};

#endif  // ENVSCONFPAGE_H
