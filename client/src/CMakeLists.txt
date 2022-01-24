set(RPC_INCLUDE_DIRS "../rpc/")
set(PAGE_HEAD_DIR "pages/container/")
set(DATA_DIR "${CMAKE_INSTALL_PREFIX}/${CMAKE_INSTALL_DATADIR}")
set(TRANSLATIONS_DIR "${DATA_DIR}/KsC-mCube/translations/")
set(TS_FILES "../translations/KsC-mCube.zh_CN.ts" )

include_directories(include ${RPC_INCLUDE_DIRS})
file(GLOB_RECURSE rpc_src "../rpc/*.cc")
file(GLOB_RECURSE cur_src "*.cpp")
file(GLOB_RECURSE HEADERS "*.h")
file(GLOB_RECURSE FORMS "*.ui")
file(GLOB_RECURSE RESOURCES "../data/*.qrc")
file(GLOB_RECURSE RESOURCES "../data/*.qrc")

add_definitions(-std=gnu++11)

qt5_create_translation( QM_FILES ${CMAKE_SOURCE_DIR} ${TS_FILES} )

add_executable(ksc-mcube-client
	${cur_src} 
	${rpc_src} 
	${RESOURCES} 
	${HEADERS} 
        ${FORMS}
        ${QM_FILES})


target_include_directories(ksc-mcube-client PRIVATE
        ${CMAKE_BINARY_DIR}
        ${KLOG_QT5_INCLUDE_DIRS}
        ${PAGE_HEAD_DIR})

target_link_libraries(ksc-mcube-client
        grpc++
        grpc++_reflection
        gpr
        protobuf
        Qt5::Widgets
        ${KLOG_QT5_LIBRARIES})

install(FILES ${QM_FILES}
    DESTINATION ${TRANSLATIONS_DIR})
