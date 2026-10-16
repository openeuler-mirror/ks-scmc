#! /usr/bin/python
from ctypes import *


lib = CDLL("/root/work/ks-scmc/client/subscribe/libsubscribe.so")


class WargLog(Structure):
    _fields_ = [
        ("id", c_long),
        ("eventType", c_long),
        ("eventModule", c_long),
        ("nodeId", c_long),
        ("nodeInfo", c_char_p),
        ("containerId", c_char_p),
        ("containerName", c_char_p),
        ("detail", c_char_p),
        ("haveRead", c_long),
        ("createdAt", c_long),
        ("updatedAt", c_long),
    ]


lib.Subscribe(c_char_p(b"127.0.0.1:10050"),
              c_char_p(b"sysadm"), c_char_p(b"12345678"))

lib.GetMessage.restype = WargLog
var = 1
while var == 1:
    logs = lib.GetMessage()
    print(logs)
