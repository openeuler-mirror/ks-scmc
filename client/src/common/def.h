#ifndef DEF_H
#define DEF_H
#include <QString>
enum GUIDE_ITEM_TYPE
{
    GUIDE_ITEM_TYPE_NORMAL,
    GUIDE_ITEM_TYPE_GROUP,
    GUIDE_ITEM_TYPE_SUB
};

enum GUIDE_ITEM
{
    GUIDE_ITEM_HONE = 0,
    GUIDE_ITEM_AUDIT_APPLY_LIST = 2,
    GUIDE_ITEM_AUDIT_WARNING_LIST = 3,
    GUIDE_ITEM_AUDIT_LOG_LIST,
    GUIDE_ITEM_CONTAINER_LIST = 6,
    GUIDE_ITEM_CONTAINER_TEMPLATE,
    GUIDE_ITEM_IMAGE_MANAGER,
    GUIDE_ITEM_NODE_MANAGER
};

enum ACTION_BUTTON_TYPE
{
    ACTION_BUTTON_TYPE_MONITOR,
    ACTION_BUTTON_TYPE_EDIT,
    ACTION_BUTTON_TYPE_TERINAL,
    ACTION_BUTTON_TYPE_MENU
};

struct CPUInfo
{
    qint64 totalCore;
    qint64 schedulingPriority;
};

struct MemoryInfo
{
    qint64 softLimit;
    qint64 maxLimit;
};

struct NetworkInfo
{
};

struct EnvsInfo
{
};

struct GraphicInfo
{
};

struct VolumesInfo
{
};

struct HighAvailabilityInfo
{
};
#endif  // DEF_H
