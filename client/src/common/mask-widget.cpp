#include "mask-widget.h"
#include <kiran-log/qt5-log-i.h>
#include <QGridLayout>
#include <QMouseEvent>
#include <QMovie>
MaskWidget::MaskWidget(QWidget *parent) : QWidget(parent), m_labLoading(nullptr)
{
    initUI();
}

void MaskWidget::setMaskVisible(bool visible)
{
    KLOG_INFO() << "setMaskVisible";
    this->setVisible(visible);
}

bool MaskWidget::maskIsVisible()
{
    return this->isVisible();
}

MaskWidget::~MaskWidget()
{
}

void MaskWidget::initUI()
{
    this->setWindowFlags(Qt::CustomizeWindowHint | Qt::FramelessWindowHint);

    QGridLayout *gridLayout = new QGridLayout(this);
    gridLayout->setMargin(0);
    gridLayout->setSpacing(0);

    m_labLoading = new QLabel(this);
    m_labLoading->setFixedSize(128, 128);
    QMovie *movie;
    movie = new QMovie(":/images/loading.gif");
    m_labLoading->setMovie(movie);
    movie->start();

    gridLayout->addWidget(m_labLoading, 0, 0, Qt::AlignCenter);

    hide();
}
