#ifndef GRAPHICCONFPAGE_H
#define GRAPHICCONFPAGE_H

#include <QWidget>
#include "common/info-worker.h"
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
    void getGraphicInfo(container::CreateRequest *request);

private:
    Ui::GraphicConfPage *ui;
};

#endif  // GRAPHICCONFPAGE_H
