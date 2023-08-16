#include "configview.h"
#include <QDebug>
#include "common/message-dialog.h"
#include "configgroup.h"
#include "operatewidget.h"

ConfigView::ConfigView(Qt::Orientation orientation, QWidget *parent) : QHeaderView(orientation, parent)
{
}

ConfigDelegate::ConfigDelegate(ConfigTableType whichTable, QWidget *parent) : QStyledItemDelegate(parent),
                                                                              m_nWidth(25),
                                                                              m_nHeight(25),
                                                                              m_ChooseTable(whichTable)
{
    m_listPage.reserve(10);
    m_listMode << tr("ReadOnly")
               << tr("ReadWrite");
}

ConfigDelegate::~ConfigDelegate()
{
    dealMemberVar();
}

QWidget *ConfigDelegate::createEditor(QWidget *parent, const QStyleOptionViewItem &option, const QModelIndex &index) const
{
    Q_UNUSED(option);
    const int &row = index.row();
    int curPage = 0;
    if (m_listPage.size() > row)
        curPage = m_listPage[row];

    if (index.column() == FIRST_COLUMN_HEADER || index.column() == SECOND_COLUMN_HEADER)
    {
        QSharedPointer<QLineEdit> pEdit(new QLineEdit(parent));
        pEdit->setFocus();
        if (index.column() == FIRST_COLUMN_HEADER)
            m_pEditFirst.append(pEdit);
        else
            m_pEditSecond.append(pEdit);
        if (curPage)
            pEdit->setDisabled(true);
        pEdit->setStyleSheet("QLineEdit{border:1px solid #E4E7ED;}");
        return pEdit.get();
    }
    else if (index.column() == THIRD_COLUMN_HEADER)
    {
        QSharedPointer<QComboBox> pBox(new QComboBox(parent));
        for (int i = 0; i < m_listMode.count(); ++i)
        {
            pBox->addItem(m_listMode[i]);
        }
        m_pComboBoxMode.append(pBox);
        if (curPage)
            pBox->setDisabled(true);
        return pBox.get();
    }
    else if (index.column() == OPERATE_HEADER)
    {
        OperateWidget *pWidget = new OperateWidget(row, curPage, parent);
        dealOperateSig(pWidget);
        //pWidget->setFocus();
        return pWidget;
    }

    return nullptr;
}

void ConfigDelegate::setEditorData(QWidget *editor, const QModelIndex &index) const
{
    Q_UNUSED(editor);
    int Curcolumn = index.column();
    if (Curcolumn == FIRST_COLUMN_HEADER || Curcolumn == SECOND_COLUMN_HEADER)
    {
        QLineEdit *pEdit = static_cast<QLineEdit *>(editor);
        pEdit->setText(index.model()->data(index, Qt::EditRole).toString());
    }
    else if (Curcolumn == THIRD_COLUMN_HEADER)
    {
        int value = index.model()->data(index, Qt::EditRole).toInt();
        QComboBox *pBox = static_cast<QComboBox *>(editor);
        pBox->setCurrentIndex(value);
    }
}

void ConfigDelegate::setModelData(QWidget *editor, QAbstractItemModel *model, const QModelIndex &index) const
{
    int Curcolumn = index.column();
    if (Curcolumn == FIRST_COLUMN_HEADER || Curcolumn == SECOND_COLUMN_HEADER)
    {
        QLineEdit *pEdit = static_cast<QLineEdit *>(editor);
        model->setData(index, pEdit->text(), Qt::EditRole);
    }
    else if (Curcolumn == THIRD_COLUMN_HEADER)
    {
        QComboBox *pBox = static_cast<QComboBox *>(editor);
        model->setData(index, pBox->currentIndex(), Qt::EditRole);
    }
}

void ConfigDelegate::updateEditorGeometry(QWidget *editor, const QStyleOptionViewItem &option, const QModelIndex &index) const
{
    Q_UNUSED(index);
    editor->setGeometry(option.rect);
}

void ConfigDelegate::dealOperateSig(const OperateWidget *pWidget) const
{
    connect(pWidget, SIGNAL(sendSaveSig(OperateWidget *)), this, SLOT(sendSaveSlot(OperateWidget *)));
    connect(pWidget, SIGNAL(sendEditSig(OperateWidget *)), this, SLOT(sendEditSlot(OperateWidget *)));
    connect(pWidget, SIGNAL(sendAddSig(OperateWidget *)), this, SLOT(sendAddSlot(OperateWidget *)));
    connect(pWidget, SIGNAL(sendDeleteSig(OperateWidget *)), this, SLOT(sendDeleteSlot(OperateWidget *)));
}

void ConfigDelegate::dealMemberVar()
{
    foreach (QSharedPointer<QLineEdit> pEdit, m_pEditFirst)
    {
        if (pEdit)
        {
            m_pEditFirst.removeOne(pEdit);
        }
    }

    foreach (QSharedPointer<QLineEdit> pEdit, m_pEditSecond)
    {
        if (pEdit)
        {
            m_pEditSecond.removeOne(pEdit);
        }
    }

    foreach (QSharedPointer<QComboBox> pBox, m_pComboBoxMode)
    {
        if (pBox)
        {
            m_pComboBoxMode.removeOne(pBox);
        }
    }
}

void ConfigDelegate::sendSaveSlot(OperateWidget *pCurWidget)
{
    int row = pCurWidget->getCurRow();
    if (m_pEditFirst.size() <= row || m_pEditSecond.size() <= row)
        return;

    if (m_pEditFirst[row]->text().isEmpty())
    {
        std::string str = "Please enter variable";
        if (CONFIG_TABLE_TYPE_VOLUMES == m_ChooseTable)
            str = "Please enter container path";
        MessageDialog::message(tr("Save Data"),
                               tr("Data is empty!"),
                               tr(str.data()),
                               ":/images/warning.png",
                               MessageDialog::StandardButton::Ok);
        return;
    }
    if (CONFIG_TABLE_TYPE_VOLUMES == m_ChooseTable && m_pEditSecond[row]->text().isEmpty())
    {
        std::string str = "Please enter host path";
        MessageDialog::message(tr("Save Data"),
                               tr("Data is empty!"),
                               tr(str.data()),
                               ":/images/warning.png",
                               MessageDialog::StandardButton::Ok);
        return;
    }

    pCurWidget->setPage(1);
    if (m_pEditFirst.size() > row)
        m_pEditFirst[row]->setDisabled(true);
    if (m_pEditSecond.size() > row)
        m_pEditSecond[row]->setDisabled(true);
    if (m_pEditSecond.size() > row)
        m_pComboBoxMode[row]->setDisabled(true);

    if (m_listPage.size() <= row)
        m_listPage.insert(row, pCurWidget->getCurPage());
    else
        m_listPage[row] = pCurWidget->getCurPage();

    emit sendSaveSig(row);
}

void ConfigDelegate::sendEditSlot(OperateWidget *pCurWidget)
{
    int row = pCurWidget->getCurRow();
    qDebug() << __func__ << __LINE__ << ", row:" << row << ", var:" << m_pEditFirst.size() << ", val:" << m_pEditSecond.size();

    if (m_pEditFirst.size() > row)
        m_pEditFirst[row]->setDisabled(false);

    if (m_pEditSecond.size() > row)
        m_pEditSecond[row]->setDisabled(false);

    if (m_pComboBoxMode.size() > row)
        m_pComboBoxMode[row]->setDisabled(false);

    if (m_listPage.size() <= row)
        m_listPage.insert(row, pCurWidget->getCurPage());
    else
        m_listPage.replace(row, pCurWidget->getCurPage());

    emit sendEditSig(row);
}

void ConfigDelegate::sendAddSlot(OperateWidget *pCurWidget)
{
    int row = pCurWidget->getCurRow();
    dealMemberVar();
    m_listPage.insert(row + 1, 0);
    emit sendAddSig(row);
}

void ConfigDelegate::sendDeleteSlot(OperateWidget *pCurWidget)
{
    int row = pCurWidget->getCurRow();
    if (m_listPage.size() == 1 || m_pEditFirst.size() == 1 || m_pComboBoxMode.size() == 1)
        return;

    dealMemberVar();
    m_listPage.removeAt(row);
    emit sendDeleteSig(row);
}
