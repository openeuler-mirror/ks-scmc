#ifndef HEADERVIEW_H
#define HEADERVIEW_H

#include <QHeaderView>
#include <QWidget>
class HeaderView : public QHeaderView
{
    Q_OBJECT
public:
    explicit HeaderView(bool isCheckable, QWidget *parent = nullptr);
    void setSortableCols(QList<int> cols);

protected:
    void paintSection(QPainter *painter, const QRect &rect, int logicalIndex) const override;
    void mousePressEvent(QMouseEvent *e) override;

signals:
    void ckbToggled(bool checked);

public slots:
    void setCheckState(bool checked);

private:
    bool m_ckbIsToggled;  //是否选中
    bool m_isCheckable = false;
    QPoint m_mousePoint;                 //鼠标位置
    mutable QRect m_rectHeaderCheckBox;  //复选框的位置
    QList<int> m_sortableCol;
};

#endif  // HEADERVIEW_H
