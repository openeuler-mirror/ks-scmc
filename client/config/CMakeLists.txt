#添加ks-scmc.desktop.in文件
configure_file (
        "${PROJECT_SOURCE_DIR}/config/ks-scmc.desktop.in"
        "${PROJECT_BINARY_DIR}/config/ks-scmc.desktop"
)

install(FILES "${PROJECT_BINARY_DIR}/config/ks-scmc.desktop"
        DESTINATION ${DATA_DIR}/applications/ )
