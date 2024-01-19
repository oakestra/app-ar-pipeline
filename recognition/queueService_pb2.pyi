from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Frame(_message.Message):
    __slots__ = ["client", "id", "qos", "data"]
    CLIENT_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    QOS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    client: str
    id: str
    qos: str
    data: bytes
    def __init__(self, client: _Optional[str] = ..., id: _Optional[str] = ..., qos: _Optional[str] = ..., data: _Optional[bytes] = ...) -> None: ...
