#include "configmodel.h"
#include <QDebug>
#include "operatewidget.h"

ConfigModel::ConfigModel(QStringList &head)
{
    m_listHead = head;
}

ConfigModel::~ConfigModel()
{
    foreach (QSharedPointer<ModelItem> pItem, m_listData)
    {
        if (pItem)
        {
            m_listData.removeOne(pItem);
        }
    }
}

Qt::ItemFlags ConfigModel::flags(const QModelIndex &index) const
{
    if (!index.isValid())
        return Qt::NoItemFlags;

    return Qt::ItemIsEnabled | Qt::ItemIsSelectable | Qt::ItemIsEditable;
}

int ConfigModel::rowCount(const QModelIndex &parent) const
{
    if (parent.isValid())
        return 0;

    return m_listData.count();
}

int ConfigModel::columnCount(const QModelIndex &parent) const
{
    if (parent.isValid())
        return 0;

    return m_listHead.count();
}

QVariant ConfigModel::headerData(int section, Qt::Orientation orientation, int role) const
{
    if (section >= m_listHead.size())
    {
        qDebug() <<__func__ << __LINE__ << ", section:" << section << ", size:" << m_listHead.size();
        return QVariant();
    }

    if (orientation == Qt::Horizontal && role == Qt::DisplayRole)
        return m_listHead[section];

    return QVariant();
}

QVariant ConfigModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid())
        return QVariant();

    if (role == Qt::TextAlignmentRole)
        return Qt::AlignCenter;

    int nRow = index.row();
    int nCol = index.column();
    if (nRow >= m_listData.count())
        return QVariant();

    if (role == Qt::EditRole)
    {
        switch (nCol)
        {
        case FIRST_COLUMN_HEADER:
            return m_listData.at(nRow)->m_firstColVal;
        case SECOND_COLUMN_HEADER:
            return m_listData[nRow]->m_secondColVal;
        case THIRD_COLUMN_HEADER:
            return m_listData[nRow]->m_thirdColVal;
        default:
            return QVariant();
        }
    }

    return QVariant();
}

bool ConfigModel::setData(const QModelIndex &index, const QVariant &value, int role)
{
    if (role != Qt::EditRole || !value.isValid() || !index.isValid() || index.row() >= m_listData.count())
    {
        qDebug()<<__func__<<__LINE__<<"param err";
        return false;
    }

    int curRow = index.row();
    auto pRowInfo = m_listData.at(curRow);
    switch (index.column())
    {
    case FIRST_COLUMN_HEADER:
        pRowInfo->m_firstColVal = value.toString();
        break;
    case SECOND_COLUMN_HEADER:
        pRowInfo->m_secondColVal = value.toString();
        break;
    case THIRD_COLUMN_HEADER:
        pRowInfo->m_thirdColVal = value.toInt();
        break;
    default:
        return false;
    }

    m_listData.replace(curRow, pRowInfo);
    return true;
}

void ConfigModel::addModelRow(QSharedPointer<ModelItem> pItem)
{
    m_listData.append(pItem);
    ShowTableinfo();
}

void ConfigModel::insertModelByRow(int Row, QSharedPointer<ModelItem> pItem)
{
    m_listData.insert(Row, pItem);
    ShowTableinfo();
}

void ConfigModel::deleteModelByRow(int Row)
{
    if (Row >= m_listData.size())
        return;

    auto pItem = m_listData[Row];
    if (!pItem)
        return;

    m_listData.removeAt(Row);
    ShowTableinfo();
}

void ConfigModel::ModifyModelByRow(int Row, QSharedPointer<ModelItem> pItem)
{
    m_listData.replace(Row, pItem);
    ShowTableinfo();
}

QSharedPointer<ModelItem> ConfigModel::getModelByRow(int Row)
{
    return m_listData.at(Row);
}

QList<QSharedPointer<ModelItem> > ConfigModel::getAllModel()
{
    return m_listData;
}

void ConfigModel::ShowTableinfo()
{
    beginResetModel();
    endResetModel();
}

