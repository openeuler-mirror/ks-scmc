#ifndef MASKWIDGET_H
#define MASKWIDGET_H

#include <QLabel>
#include <QObject>
class MaskWidget : public QWidget
{
    Q_OBJECT
public:
    explicit MaskWidget(QWidget *parent = nullptr);
    void setMaskVisible(bool visible);
    bool maskIsVisible();
    ~MaskWidget();

private:
    void initUI();

signals:

public slots:

private:
    QLabel *m_labLoading;
};

#endif  // MASKWIDGET_H
