#include "graphic-conf-page.h"
#include "ui_graphic-conf-page.h"

GraphicConfPage::GraphicConfPage(QWidget *parent) :
    QWidget(parent),
    ui(new Ui::GraphicConfPage)
{
    ui->setupUi(this);
}

GraphicConfPage::~GraphicConfPage()
{
    delete ui;
}
