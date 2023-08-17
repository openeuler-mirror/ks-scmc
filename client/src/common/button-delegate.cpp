#include "button-delegate.h"
#include <QApplication>
#include <QMouseEvent>
#include "def.h"
#define BUTTON_WIDTH 15
#define BUTTON_HEIGHT 15
#define BUTTON_SPACE 20
#define BUTTON_TOP 17

ButtonDelegate::ButtonDelegate(QMap<int, QString> btnInfo, QObject *parent)
    : QStyledItemDelegate(parent),
      m_btnInfo(btnInfo),
      m_menu(nullptr)
{
    if (!m_btnInfo.value(ACTION_BUTTON_TYPE_MENU).isNull())
    {
        m_menu = new QMenu();
        m_menu->setObjectName("moreInTableMenu");
        m_menu->addAction(tr("Run"));
        m_menu->addAction(tr("Stop"));
        m_menu->addAction(tr("Restart"));
        connect(m_menu, &QMenu::triggered, this, &ButtonDelegate::onActTriggered);
    }
}

ButtonDelegate::~ButtonDelegate()
{
    if (m_menu)
    {
        delete m_menu;
        m_menu = nullptr;
    }
}

void ButtonDelegate::paint(QPainter *painter, const QStyleOptionViewItem &option, const QModelIndex &index) const
{
    QStyleOptionViewItem viewOption(option);
    initStyleOption(&viewOption, index);
    if (option.state.testFlag(QStyle::State_HasFocus))
        viewOption.state = viewOption.state ^ QStyle::State_HasFocus;

    QStyledItemDelegate::paint(painter, viewOption, index);
    int count = 0;

    QMap<int, QString>::const_iterator i = m_btnInfo.constBegin();
    while (i != m_btnInfo.constEnd())
    {
        // 绘制按钮
        QStyleOptionButton button;
        QRect btnRect = QRect(option.rect.x() + BUTTON_SPACE + count * BUTTON_WIDTH + count * BUTTON_SPACE, option.rect.y() + BUTTON_TOP, BUTTON_WIDTH, BUTTON_HEIGHT);
        button.state |= QStyle::State_Enabled;
        if (btnRect.contains(m_mousePoint))
        {
            if (m_nType == 0)
            {
                button.state |= QStyle::State_MouseOver;
            }
            else if (m_nType == 1)
            {
                button.state |= QStyle::State_Sunken;
            }
        }
        QPixmap pixBtn;
        pixBtn.load(i.value());
        QApplication::style()->drawItemPixmap(painter, btnRect, Qt::AlignHCenter | Qt::AlignVCenter, pixBtn);
        ++i;
        count++;
    }
}

bool ButtonDelegate::editorEvent(QEvent *event, QAbstractItemModel *model, const QStyleOptionViewItem &option, const QModelIndex &index)
{
    Q_UNUSED(model);
    Q_UNUSED(index);
    QMouseEvent *pEvent = static_cast<QMouseEvent *>(event);
    m_mousePoint = pEvent->pos();
    QApplication::restoreOverrideCursor();

    bool repaint = false;
    int count = 0;
    QMap<int, QString>::const_iterator i = m_btnInfo.constBegin();
    while (i != m_btnInfo.constEnd())
    {
        // 绘制按钮
        QStyleOptionButton button;
        QRect btnRect = QRect(option.rect.x() + BUTTON_SPACE + count * BUTTON_WIDTH + count * BUTTON_SPACE, option.rect.y() + BUTTON_TOP, BUTTON_WIDTH, BUTTON_HEIGHT);
        // 鼠标位于按钮之上
        if (!btnRect.contains(m_mousePoint))
        {
            i++;
            count++;
            continue;
        }
        repaint = true;
        switch (event->type())
        {
        // 鼠标滑过
        case QEvent::MouseMove:
        {
            // 设置鼠标样式为手型
            QApplication::setOverrideCursor(Qt::PointingHandCursor);
            m_nType = 0;
            break;
        }
        // 鼠标按下
        case QEvent::MouseButtonPress:
        {
            m_nType = 1;
            break;
        }
        // 鼠标释放
        case QEvent::MouseButtonRelease:
        {
            switch (i.key())
            {
            case ACTION_BUTTON_TYPE_MONITOR:
            {
                emit sigMonitor(index.row());
                break;
            }
            case ACTION_BUTTON_TYPE_EDIT:
            {
                emit sigEdit(index.row());
                break;
            }
            case ACTION_BUTTON_TYPE_TERINAL:
            {
                emit sigTerminal(index.row());
                break;
            }
            case ACTION_BUTTON_TYPE_MENU:
            {
                if (m_menu)
                {
                    m_index = index;
                    QPoint point(QCursor::pos().x() + 10, QCursor::pos().y());
                    m_menu->popup(point);
                }
            }
            default:
                break;
            }
        }
        default:
            break;
        }
        count++;
        i++;
    }
    return repaint;
}

void ButtonDelegate::onActTriggered(QAction *act)
{
    if (act->text() == tr("Run"))
        emit sigActRun(m_index);
    else if (act->text() == tr("Stop"))
        emit sigActStop(m_index);
    else if (act->text() == tr("Restart"))
        emit sigActRestart(m_index);
}
