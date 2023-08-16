#ifndef NODEADDITION_H
#define NODEADDITION_H

#include <QWidget>

namespace Ui
{
class NodeAddition;
}

class NodeAddition : public QWidget
{
    Q_OBJECT

public:
    explicit NodeAddition(QWidget *parent = nullptr);
    ~NodeAddition();

signals:
    void sigSave(QMap<QString, QString> newNodeInfo);

private slots:
    void onSave();
    void onCancel();

private:
    Ui::NodeAddition *ui;
};

#endif  // NODEADDITION_H
