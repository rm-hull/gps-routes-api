import hashlib


def get_object_id(ref: str) -> str:
    m = hashlib.md5()
    m.update(ref.encode("utf-8"))
    return m.hexdigest()