project(client)

cmake_minimum_required(VERSION 2.8)

include(GNUInstallDirs)

set(CMAKE_INCLUDE_CURRENT_DIR ON)

set(CMAKE_AUTOUIC ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

#使用编译时的链接路径
set(CMAKE_INSTALL_RPATH_USE_LINK_PATH TRUE)
set(DATA_DIR "${CMAKE_INSTALL_PREFIX}/${CMAKE_INSTALL_DATADIR}")
set(BIN_DIR "${CMAKE_INSTALL_PREFIX}/${CMAKE_INSTALL_BINDIR}")
set(ICON_DIR "${DATA_DIR}/ks-scmc/icons")
set(TRANSLATIONS_DIR "${DATA_DIR}/ks-scmc/translations/")
# QtCreator supports the following variables for Android, which are identical to qmake Android variables.
# Check http://doc.qt.io/qt-5/deployment-android.html for more information.
# They need to be set before the find_package(Qt5 ...) call.

find_package(PkgConfig REQUIRED)
find_package(Qt5 COMPONENTS Widgets REQUIRED LinguistTools)
pkg_search_module(KLOG_QT5 REQUIRED klog-qt5)

#添加config.h.in文件
configure_file (
    "${PROJECT_SOURCE_DIR}/config/config.h.in"
    "${PROJECT_BINARY_DIR}/config/config.h"
)

add_subdirectory(src)
add_subdirectory(config)
add_subdirectory(data/images)
