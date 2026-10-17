import grpc

import logging_pb2, logging_pb2_grpc


class RpcClient(object):
    def __init__(self, addr, name, token):
        self._addr = addr
        self._md = (
            ('x-api-client-name', name),
            ('x-api-client-token', token),
        )

    def _chan(self):
        return grpc.insecure_channel(self._addr)

    def query_runtime_log(self, **kwargs):
        """ 查询审计事件日志 """
        with self._chan() as chan:
            stub = logging_pb2_grpc.LoggingStub(chan)
            return stub.ListRuntime(
                logging_pb2.ListRuntimeRequest(**kwargs),
                metadata=self._md)

    def query_warn_log(self, **kwargs):
        """ 查询告警事件日志 """
        with self._chan() as chan:
            stub = logging_pb2_grpc.LoggingStub(chan)
            return stub.ListWarn(
                logging_pb2.ListWarnRequest(**kwargs),
                metadata=self._md)

    def set_warn_log_read(self, ids):
        """ 设置告警事件日志已读 """
        with self._chan() as chan:
            stub = logging_pb2_grpc.LoggingStub(chan)
            return stub.ReadWarn(
                logging_pb2.ReadWarnRequest(ids=ids),
                metadata=self._md)

    def subscribe_warn_log(self):
        """ 实时订阅告警事件日志 """
        chan = self._chan()
        stub = logging_pb2_grpc.LoggingStub(chan)
        return stub.SubscribeWarn(
            logging_pb2.SubscribeWarnRequest(),
            metadata=self._md)
