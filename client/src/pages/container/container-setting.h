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
    void setContainerNodeIds(QPair<int64_t, QString> ids);

protected:
    bool eventFilter(QObject *obj, QEvent *ev);

private:
    void initUI();
    void initSummaryUI();
    GuideItem *createGuideItem(QListWidget *parent, QString text, int type = GUIDE_ITEM_TYPE_NORMAL, QString icon = "");
    void initBaseConfPages();
    void initAdvancedConfPages();
    void updateRemovableItem(QString itemText);
    void getNodeInfo();
    void getImageInfo(int64_t node_id);
    void createContainer();
    void updateContainer();

signals:
    void sigUpdateContainer();

private slots:
    void onItemClicked(QListWidgetItem *item);
    void onAddItem(QAction *action);
    void onDelItem();
    void onConfirm();
    void onNodeSelectedChanged(QString newStr);
    void getNodeListResult(const QPair<grpc::Status, node::ListReply> &);
    void getCreateContainerResult(const QPair<grpc::Status, container::CreateReply> &);
    void getContainerInspectResult(const QPair<grpc::Status, container::InspectReply> &);
    void getUpdateContainerResult(const QPair<grpc::Status, container::UpdateReply> &);
    void getListImageFinishedResult(const QPair<grpc::Status, image::ListReply> &);

private:
    Ui::ContainerSetting *ui = nullptr;
    QStackedWidget *m_baseConfStack = nullptr;
    QStackedWidget *m_advancedConfStack = nullptr;
    QList<GuideItem *> m_baseItemMap;
    QList<GuideItem *> m_advancedItemMap;
    QMenu *m_addMenu = nullptr;
    int m_netWorkCount;
    ContainerSettingType m_type;
    QComboBox *m_cbImage = nullptr;
    QLabel *m_labImage = nullptr;
    QMap<int64_t, QString> m_nodeInfo;
    QPair<int64_t, QString> m_containerIds;
    double m_totalCPU = 0.0;
};

#endif  // CONTAINERSETTING_H
