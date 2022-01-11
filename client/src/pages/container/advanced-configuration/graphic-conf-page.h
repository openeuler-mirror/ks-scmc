#ifndef GRAPHICCONFPAGE_H
#define GRAPHICCONFPAGE_H

#include <QWidget>

namespace Ui
{
class GraphicConfPage;
}

class GraphicConfPage : public QWidget
{
    Q_OBJECT

public:
    explicit GraphicConfPage(QWidget *parent = nullptr);
    ~GraphicConfPage();
    void setGraphicInfo();

private:
    Ui::GraphicConfPage *ui;
};

#endif  // GRAPHICCONFPAGE_H
