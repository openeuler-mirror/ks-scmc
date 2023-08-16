#ifndef CONFIGMODEL_H
#define CONFIGMODEL_H

#include <QAbstractTableModel>
#include <QStringList>
#include "configgroup.h"

class ConfigModel : public QAbstractTableModel
{
    Q_OBJECT
public:
    explicit ConfigModel(QStringList &head);
    ~ConfigModel();
    Qt::ItemFlags flags(const QModelIndex &index) const override;
    int rowCount(const QModelIndex &parent = QModelIndex()) const override;
    int columnCount(const QModelIndex &parent = QModelIndex()) const override;
    QVariant headerData(int section, Qt::Orientation orientation, int role) const override;
    QVariant data(const QModelIndex &index, int role = Qt::DisplayRole) const override;
    bool setData(const QModelIndex &index, const QVariant &value, int role = Qt::EditRole) override;
    void addModelRow(QSharedPointer<ModelItem> pItem);
    void insertModelByRow(int Row, QSharedPointer<ModelItem> pItem);
    void deleteModelByRow(int Row);
    void ModifyModelByRow(int Row, QSharedPointer<ModelItem> pItem);
    QSharedPointer<ModelItem> getModelByRow(int Row);
    QList<QSharedPointer<ModelItem>> getAllModel();

private:
    void ShowTableinfo();

private:
    QStringList m_listHead;
    QList<QSharedPointer<ModelItem>> m_listData;
};

#endif  // CONFIGMODEL_H
