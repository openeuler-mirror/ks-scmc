set(RPC_INCLUDE_DIRS "../rpc/")
set(GRPC_LIBRARIES "/usr/lib64/libgrpc++.so.1.31.0")
set(GRPC_REFLECTION_LIBRARIES "/usr/lib64/libgrpc++_reflection.so.1.31.0")
set(PROTOBUF_LIBRARIES "/usr/lib64/libprotobuf.so.23.0.3")
set(PAGE_HEAD_DIR "pages/container/")

file(GLOB_RECURSE RESOURCES "../data/*.qrc")

include_directories(include ${RPC_INCLUDE_DIRS})
file(GLOB_RECURSE rpc_src "../rpc/*.cc")
file(GLOB_RECURSE cur_src "*.cpp")
file(GLOB_RECURSE HEADERS "*.h")
file(GLOB_RECURSE FORMS "*.ui")
file(GLOB_RECURSE RESOURCES "../data/*.qrc")

add_definitions(-std=gnu++11)

add_executable(grpc-client 
	${cur_src} 
	${rpc_src} 
	${RESOURCES} 
	${HEADERS} 
        ${FORMS})


target_include_directories(grpc-client PRIVATE
        ${CMAKE_BINARY_DIR}
        ${KLOG_QT5_INCLUDE_DIRS}
        ${PAGE_HEAD_DIR})

target_link_libraries(grpc-client 
	${GRPC_LIBRARIES} 
	${GRPC_REFLECTION_LIBRARIES} 
	${PROTOBUF_LIBRARIES}
	Qt5::Widgets
	${KLOG_QT5_LIBRARIES})
