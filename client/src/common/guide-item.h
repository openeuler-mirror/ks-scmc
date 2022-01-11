#ifndef GUIDEITEM_H
#define GUIDEITEM_H
#include <QListWidgetItem>
#include <QToolButton>
#include <QWidget>
#include "common/def.h"
namespace Ui
{
class GuideItem;
}

enum TipLinePosition
{
    TIP_LINE_POSITION_LEFT,
    TIP_LINE_POSITION_RIGHT
};

class GuideItem : public QWidget
{
    Q_OBJECT

public:
    explicit GuideItem(QString text, QString icon = "", int type = GUIDE_ITEM_TYPE_NORMAL, QWidget *parent = nullptr);
    ~GuideItem();
    void setArrow(bool isHide);
    void setTipLinePosition(TipLinePosition position);
    void setSelected(bool isSelected);
    void setDeleteBtn();
    void removeDeleteBtn();
    void setDeleteBtnVisible(bool visible);
    int getItemType();
    QString getItemText();

private:
    void initUI(QString text, QString lIcon, int type);

protected:
    void paintEvent(QPaintEvent *event) override;
    void enterEvent(QEvent *event) override;
    void leaveEvent(QEvent *event) override;

private slots:
    void onDelete();
signals:
    void sigDeleteItem();

private:
    Ui::GuideItem *ui;
    int m_type = 0;
    bool m_isSelected = false;
    bool m_delBtnVisible = false;
    QToolButton *m_delBtn;
    TipLinePosition m_position;
    QString m_text;
};

#endif  // GUIDEITEM_H
