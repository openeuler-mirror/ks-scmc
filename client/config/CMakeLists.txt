#添加ksc-mcube.desktop.in文件
configure_file (
        "${PROJECT_SOURCE_DIR}/config/ksc-mcube.desktop.in"
        "${PROJECT_BINARY_DIR}/config/ksc-mcube.desktop"
)

install(FILES "${PROJECT_BINARY_DIR}/config/ksc-mcube.desktop"
        DESTINATION ${DATA_DIR}/applications/ )
