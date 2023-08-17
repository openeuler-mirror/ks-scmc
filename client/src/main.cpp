#include <kiran-log/qt5-log-i.h>
#include <QApplication>
#include <QDesktopWidget>
#include <QFile>
#include <QMessageBox>
#include <QStyle>
#include <QTranslator>
#include <iostream>
#include "config/config.h"
#include "main-window.h"

#define TRANSLATION_DIR TRANSLATIONS_FILE_DIR
int main(int argc, char *argv[])
{
    //设置日志输出
    if (klog_qt5_init("", "kylinsec-session", "ksc-mcube-client", "ksc-mcube-client") < 0)
    {
        std::cout << "init klog error" << std::endl;
    }
    KLOG_INFO("******New Output*********\n");

    QApplication a(argc, argv);

    ///加载qss样式表
    QFile file(":/style/theme.qss");
    if (file.open(QFile::ReadOnly))
    {
        QString styleSheet = QLatin1String(file.readAll());

        a.setStyleSheet(styleSheet);
        file.close();
    }
    else
    {
        QMessageBox::warning(NULL, "warning", "Open failed", QMessageBox::Yes | QMessageBox::No, QMessageBox::Yes);
    }
    //加载翻译文件
    QTranslator *qtTranslator = new QTranslator(qApp);
    if (qtTranslator->load(QLocale(), "ksc-mcube", ".", TRANSLATION_DIR, ".qm"))
    {
        a.installTranslator(qtTranslator);
    }
    else
    {
        qDebug("Load Translator File failed : %s\n", TRANSLATION_DIR);
    }

    MainWindow w;
    int screenNum = QApplication::desktop()->screenNumber(QCursor::pos());
    QRect screenGeometry = QApplication::desktop()->screenGeometry(screenNum);
    int iTitleBarHeight = w.style()->pixelMetric(QStyle::PM_TitleBarHeight);                               // 获取标题栏高度
    w.setGeometry(0, iTitleBarHeight, screenGeometry.width(), screenGeometry.height() - iTitleBarHeight);  // 设置窗体充满桌面客户区

    w.show();
    return a.exec();
}
