#ifndef CONTAINERSETTING_H
#define CONTAINERSETTING_H
#include <QLabel>
#include <QLineEdit>
#include <QListWidgetItem>
#include <QStackedWidget>
#include <QWidget>
#include "common/def.h"
namespace Ui
{
class ContainerSetting;
}

enum TabConfigGuideItemType
{
    TAB_CONFIG_GUIDE_ITEM_TYPE_CPU,
    TAB_CONFIG_GUIDE_ITEM_TYP_MEMORY,
    TAB_CONFIG_GUIDE_ITEM_TYP_NETWORK_CARD,
    TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_ENVS,
    TAB_CONFIG_GUIDE_ITEM_TYPDE_ITEM_GRAPHIC,
    TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_VOLUMES,
    TAB_CONFIG_GUIDE_ITEM_TYP_HIGH_AVAILABILITY
};

class GuideItem;
class ContainerSetting : public QWidget
{
    Q_OBJECT

public:
    explicit ContainerSetting(QWidget *parent = nullptr);
    ~ContainerSetting();
    void paintEvent(QPaintEvent *event);
    void setItems(int row, int col, QWidget *);
    void setTitle(QString title);

protected:
    bool eventFilter(QObject *obj, QEvent *ev);

private:
    void initUI();
    GuideItem *createGuideItem(QListWidget *parent, QString text, int type = GUIDE_ITEM_TYPE_NORMAL, QString icon = "");
    void initBaseConfPages();
    void initAdvancedConfPages();
    QStringList getNodes();
    void updateRemovableItem(QString itemText);

private slots:
    void popupMessageDialog();
    void onItemClicked(QListWidgetItem *item);
    void onAddItem(QAction *action);
    void onDelItem(QWidget *sender);

private:
    Ui::ContainerSetting *ui;
    QStackedWidget *m_baseConfStack;
    QStackedWidget *m_advancedConfStack;
    QList<GuideItem *> m_baseItemMap;
    QList<GuideItem *> m_advancedItemMap;
    QMenu *m_addMenu;
    int m_netWorkCount;
};

#endif  // CONTAINERSETTING_H
