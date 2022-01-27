#include "graphic-conf-page.h"
#include "ui_graphic-conf-page.h"

GraphicConfPage::GraphicConfPage(QWidget* parent) : QWidget(parent),
                                                    ui(new Ui::GraphicConfPage)
{
    ui->setupUi(this);
}

GraphicConfPage::~GraphicConfPage()
{
    delete ui;
}

void GraphicConfPage::getGraphicInfo(container::CreateRequest* request)
{
    request->set_enable_graphic(ui->graphic_checkBox->isChecked());
}
