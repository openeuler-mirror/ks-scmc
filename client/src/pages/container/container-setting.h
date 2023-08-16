#ifndef CONTAINERSETTING_H
#define CONTAINERSETTING_H
#include <QComboBox>
#include <QLabel>
#include <QLineEdit>
#include <QListWidgetItem>
#include <QStackedWidget>
#include <QWidget>
#include "common/def.h"
#include "common/info-worker.h"
namespace Ui
{
class ContainerSetting;
}
enum ContainerSettingType
{
    CONTAINER_SETTING_TYPE_CONTAINER_CREATE,
    CONTAINER_SETTING_TYPE_CONTAINER_EDIT,
    CONTAINER_SETTING_TYPE_TEMPLATE_CREATE,
    CONTAINER_SETTING_TYPE_TEMPLATE_EDIT
};

enum TabConfigGuideItemType
{
    TAB_CONFIG_GUIDE_ITEM_TYPE_CPU = 0,
    TAB_CONFIG_GUIDE_ITEM_TYP_MEMORY,
    TAB_CONFIG_GUIDE_ITEM_TYP_NETWORK_CARD,
    TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_ENVS = 0,
    TAB_CONFIG_GUIDE_ITEM_TYPDE_ITEM_GRAPHIC,
    TAB_CONFIG_GUIDE_ITEM_TYP_ITEM_VOLUMES,
    TAB_CONFIG_GUIDE_ITEM_TYP_HIGH_AVAILABILITY
};

class GuideItem;
class ContainerSetting : public QWidget
{
    Q_OBJECT

public:
    explicit ContainerSetting(ContainerSettingType type, QWidget *parent = nullptr);
    ~ContainerSetting();
    void paintEvent(QPaintEvent *event);
    void setItems(int row, int col, QWidget *);
    void setTitle(QString title);

protected:
    bool eventFilter(QObject *obj, QEvent *ev);

private:
    void initUI();
    void initSummaryUI();
    GuideItem *createGuideItem(QListWidget *parent, QString text, int type = GUIDE_ITEM_TYPE_NORMAL, QString icon = "");
    void initBaseConfPages();
    void initAdvancedConfPages();
    QStringList getNodes();
    void updateRemovableItem(QString itemText);
    void getNodeInfo();

private slots:
    void popupMessageDialog();
    void onItemClicked(QListWidgetItem *item);
    void onAddItem(QAction *action);
    void onDelItem(QWidget *sender);
    void onConfirm();
    void onCancel();
    void onNodeSelectedChanged(QString newStr);
    void getNodeListResult(QPair<grpc::Status, node::ListReply> reply);
    void getCreateContainerResult(QPair<grpc::Status, container::CreateReply> reply);

private:
    Ui::ContainerSetting *ui;
    QStackedWidget *m_baseConfStack;
    QStackedWidget *m_advancedConfStack;
    QList<GuideItem *> m_baseItemMap;
    QList<GuideItem *> m_advancedItemMap;
    QMenu *m_addMenu;
    int m_netWorkCount;
    ContainerSettingType m_type;
    QComboBox *m_cbImage;
    QLabel *m_labImage;
    QMap<int64_t, QString> m_nodeInfo;
    double m_totalCPU = 0.0;
};

#endif  // CONTAINERSETTING_H
