#include "header-view.h"
#include <QMouseEvent>
#include <QPainter>
#include <QStandardItemModel>
#include <iostream>
using namespace std;

#define CHECKABLE_COL 0

HeaderView::HeaderView(bool isCheckable, QWidget *parent)
    : QHeaderView(Qt::Horizontal, parent),
      m_ckbIsToggled(false),
      m_isCheckable(isCheckable)
{
    m_mousePoint.setX(100);
    m_mousePoint.setY(100);
    setFixedHeight(50);
}

void HeaderView::setSortableCols(QList<int> cols)
{
    m_sortableCol = cols;
}

void HeaderView::paintSection(QPainter *painter, const QRect &rect, int logicalIndex) const
{  //保存第一列的位置
    painter->save();
    QStyleOption opt;
    //opt.init(this);
    style()->drawPrimitive(QStyle::PE_Widget, &opt, painter, this);

    QHeaderView::paintSection(painter, rect, logicalIndex);
    painter->restore();
    if (logicalIndex == 0 && m_isCheckable)  //第一列
    {
        m_rectHeaderCheckBox.setX(rect.x() + 15);
        m_rectHeaderCheckBox.setY(rect.y() + 17);
        m_rectHeaderCheckBox.setWidth(16);
        m_rectHeaderCheckBox.setHeight(16);

        QStyleOptionButton optionCheckbox;
        QPixmap pixCheckbox;

        if (m_ckbIsToggled)
        {                                              //图片需要在资源里添加
            pixCheckbox.load(":/images/checked.png");  //选中
            optionCheckbox.state = QStyle::State_On;
        }
        else
        {
            pixCheckbox.load(":/images/check-default.png");  //正常
            optionCheckbox.state = QStyle::State_Off;
        }

        style()->drawItemPixmap(painter, m_rectHeaderCheckBox, Qt::AlignLeft | Qt::AlignVCenter, pixCheckbox);
    }
}
void HeaderView::mousePressEvent(QMouseEvent *event)
{
    //表头可控制，鼠标第一个列，鼠标在矩形范围内
    if (isEnabled() && m_isCheckable && logicalIndexAt(event->pos()) == CHECKABLE_COL && m_rectHeaderCheckBox.contains(event->pos()))
    {
        m_ckbIsToggled = !m_ckbIsToggled;
        updateSection(logicalIndexAt(event->pos()));  //刷新表头第一列
        emit ckbToggled(m_ckbIsToggled);
    }
    else if (!m_sortableCol.isEmpty() && m_sortableCol.contains(logicalIndexAt(event->pos())))
    {
        Qt::SortOrder so = this->sortIndicatorOrder();
        if (so == Qt::AscendingOrder)
        {
            this->setSortIndicator(logicalIndexAt(event->pos()), Qt::DescendingOrder);
        }
        else if (so == Qt::DescendingOrder)
        {
            this->setSortIndicator(logicalIndexAt(event->pos()), Qt::AscendingOrder);
        }
    }
    QHeaderView::mousePressEvent(event);
}

void HeaderView::setCheckState(bool checked)
{
    m_ckbIsToggled = checked;
    updateSection(CHECKABLE_COL);
}
