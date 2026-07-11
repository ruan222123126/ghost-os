const RESTRICTION_TEMPLATE: &str = r#"
import builtins

_GHOST_ALLOWED_MODULES = set(__ALLOWED_MODULES_JSON__)
_GHOST_TOOLS = __PRIVATE_TOOLS_BINDING__
_GHOST_ORIGINAL_IMPORT = builtins.__import__
_GHOST_IMPORT_DEPTH = 0
_GHOST_ORIGINAL_BUILTINS = {
    "open": builtins.open,
    "eval": builtins.eval,
    "exec": builtins.exec,
    "compile": builtins.compile,
    "input": builtins.input,
}

def _ghost_normalize_open_path(file):
    if isinstance(file, str):
        return file
    fspath = getattr(file, "__fspath__", None)
    if callable(fspath):
        value = fspath()
        if isinstance(value, str):
            return value
    raise TypeError("sandbox open() only supports str and os.PathLike paths")

def _ghost_parse_open_mode(mode):
    if mode is None:
        normalized = "r"
    elif not isinstance(mode, str):
        raise TypeError("sandbox open() mode must be a string")
    else:
        normalized = mode.strip() or "r"

    if "b" in normalized:
        raise ValueError("sandbox open() does not support binary mode")
    if "+" in normalized:
        raise ValueError("sandbox open() does not support update mode")
    if "x" in normalized:
        raise ValueError("sandbox open() does not support exclusive creation mode")
    if normalized not in ("r", "rt", "tr", "w", "wt", "tw", "a", "at", "ta"):
        raise ValueError("sandbox open() supports only text modes r, w, and a")
    if "r" in normalized:
        return "r"
    if "w" in normalized:
        return "w"
    return "a"

def _ghost_normalize_encoding(encoding):
    if encoding is None:
        return "utf-8"
    if not isinstance(encoding, str):
        raise TypeError("sandbox open() encoding must be a string")
    normalized = encoding.strip().lower().replace("_", "-")
    if normalized in ("utf-8", "utf8"):
        return "utf-8"
    raise ValueError("sandbox open() only supports UTF-8 text encoding")

class _GhostSandboxFile:
    def __init__(self, path, mode, encoding):
        self.name = path
        self.mode = mode
        self.encoding = encoding
        self.closed = False
        self._cursor = 0
        self._content = ""
        self._written_chars = 0
        self._readable = mode == "r"
        self._writable = mode in ("w", "a")
        self._write_mode = None
        if self._readable:
            self._content = _GHOST_TOOLS.read_file(path=path)
        else:
            self._write_mode = "write" if mode == "w" else "append"
            _GHOST_TOOLS.write_file(path=path, content="", mode=self._write_mode)
            self._write_mode = "append"

    def _ensure_open(self):
        if self.closed:
            raise ValueError("I/O operation on closed file")

    def readable(self):
        return self._readable

    def writable(self):
        return self._writable

    def seekable(self):
        return self._readable

    def read(self, size=-1):
        self._ensure_open()
        if not self._readable:
            raise OSError("file not open for reading")
        if size is None or size < 0:
            data = self._content[self._cursor :]
            self._cursor = len(self._content)
            return data
        end = min(len(self._content), self._cursor + size)
        data = self._content[self._cursor : end]
        self._cursor = end
        return data

    def readline(self, size=-1):
        self._ensure_open()
        if not self._readable:
            raise OSError("file not open for reading")
        if self._cursor >= len(self._content):
            return ""
        newline_index = self._content.find("\n", self._cursor)
        end = len(self._content) if newline_index == -1 else newline_index + 1
        if size is not None and size >= 0:
            end = min(end, self._cursor + size)
        data = self._content[self._cursor : end]
        self._cursor = end
        return data

    def readlines(self, hint=-1):
        lines = []
        total = 0
        while True:
            line = self.readline()
            if line == "":
                break
            lines.append(line)
            total += len(line)
            if hint is not None and hint >= 0 and total >= hint:
                break
        return lines

    def seek(self, offset, whence=0):
        self._ensure_open()
        if not self._readable:
            raise OSError("file does not support seeking")
        if whence == 0:
            target = offset
        elif whence == 1:
            target = self._cursor + offset
        elif whence == 2:
            target = len(self._content) + offset
        else:
            raise ValueError("invalid whence")
        if target < 0:
            raise ValueError("negative seek position")
        self._cursor = min(target, len(self._content))
        return self._cursor

    def tell(self):
        self._ensure_open()
        if self._readable:
            return self._cursor
        return self._written_chars

    def write(self, data):
        self._ensure_open()
        if not self._writable:
            raise OSError("file not open for writing")
        if not isinstance(data, str):
            raise TypeError("write() argument must be str")
        _GHOST_TOOLS.write_file(path=self.name, content=data, mode=self._write_mode)
        self._written_chars += len(data)
        return len(data)

    def writelines(self, lines):
        self._ensure_open()
        for line in lines:
            self.write(line)

    def flush(self):
        self._ensure_open()
        return None

    def close(self):
        self.closed = True

    def isatty(self):
        self._ensure_open()
        return False

    def __iter__(self):
        self._ensure_open()
        return self

    def __next__(self):
        line = self.readline()
        if line == "":
            raise StopIteration
        return line

    def __enter__(self):
        self._ensure_open()
        return self

    def __exit__(self, exc_type, exc, tb):
        self.close()
        return False

def _ghost_restricted_import(name, globals=None, locals=None, fromlist=(), level=0):
    global _GHOST_IMPORT_DEPTH
    root = name.split(".", 1)[0]
    allowed = root in _GHOST_ALLOWED_MODULES
    if not allowed and _GHOST_IMPORT_DEPTH > 0:
        allowed = True
    if not allowed and isinstance(globals, dict):
        importer_name = str(globals.get("__name__") or "")
        importer_package = str(globals.get("__package__") or "")
        importer_scope = importer_package or importer_name
        if importer_scope and importer_name != "__main__":
            importer_root = importer_scope.split(".", 1)[0]
            if importer_root in _GHOST_ALLOWED_MODULES:
                allowed = True
        elif level > 0 and importer_scope:
            importer_root = importer_scope.split(".", 1)[0]
            allowed = importer_root in _GHOST_ALLOWED_MODULES
    if not allowed:
        raise ImportError(f"Module '{name}' is not allowed in sandbox")

    previous_builtins = {}
    for _builtin_name, _original in _GHOST_ORIGINAL_BUILTINS.items():
        previous_builtins[_builtin_name] = getattr(builtins, _builtin_name)
        setattr(builtins, _builtin_name, _original)
    _GHOST_IMPORT_DEPTH += 1
    try:
        return _GHOST_ORIGINAL_IMPORT(name, globals, locals, fromlist, level)
    finally:
        _GHOST_IMPORT_DEPTH -= 1
        for _builtin_name, _previous in previous_builtins.items():
            setattr(builtins, _builtin_name, _previous)

def _ghost_open(
    file,
    mode="r",
    buffering=-1,
    encoding=None,
    errors=None,
    newline=None,
    closefd=True,
    opener=None,
):
    if buffering != -1:
        raise ValueError("sandbox open() only supports default buffering")
    if errors not in (None, "strict"):
        raise ValueError("sandbox open() only supports strict error handling")
    if newline is not None:
        raise ValueError("sandbox open() does not support custom newline handling")
    if not closefd:
        raise ValueError("sandbox open() requires closefd=True")
    if opener is not None:
        raise ValueError("sandbox open() does not support custom opener")

    normalized_path = _ghost_normalize_open_path(file)
    normalized_mode = _ghost_parse_open_mode(mode)
    normalized_encoding = _ghost_normalize_encoding(encoding)
    return _GhostSandboxFile(normalized_path, normalized_mode, normalized_encoding)

def _ghost_blocked_builtin(name):
    def _ghost_impl(*_args, **_kwargs):
        raise PermissionError(f"builtin '{name}' is not allowed in sandbox")
    return _ghost_impl

builtins.__import__ = _ghost_restricted_import
builtins.open = _ghost_open
for _builtin_name in ("eval", "exec", "compile", "input"):
    setattr(builtins, _builtin_name, _ghost_blocked_builtin(_builtin_name))
"#;

pub(super) fn build_restriction_code(allowed_modules: &[String]) -> String {
    let allowed_modules_json =
        serde_json::to_string(allowed_modules).unwrap_or_else(|_| "[]".to_string());
    RESTRICTION_TEMPLATE
        .replace("__ALLOWED_MODULES_JSON__", &allowed_modules_json)
        .replace(
            "__PRIVATE_TOOLS_BINDING__",
            crate::sandbox::SCRIPT_EXEC_PRIVATE_TOOLS_NAME,
        )
}
