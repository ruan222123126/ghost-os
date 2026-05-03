pub(super) const STDOUT_SETUP_CODE: &str = r#"
import io
import sys
__ghost_old_stdout = sys.stdout
__ghost_stdout_capture = io.StringIO()
sys.stdout = __ghost_stdout_capture
"#;

const HELPER_BOOTSTRAP_TEMPLATE: &str = r#"
def list_files(path="."):
    return __PRIVATE_TOOLS__.list_files(path=path)

def read_file(path, start_line=None, end_line=None):
    return __PRIVATE_TOOLS__.read_file(path=path, start_line=start_line, end_line=end_line)

def search_files(query, path=".", max_results=50):
    return __PRIVATE_TOOLS__.search_files(
        query=query,
        path=path,
        max_results=max_results,
    )

def write_file(path, content, mode="write"):
    return __PRIVATE_TOOLS__.write_file(path=path, content=content, mode=mode)

def apply_diff(path, diff_text):
    return __PRIVATE_TOOLS__.apply_diff(path=path, diff_text=diff_text)

def bash_exec(command, max_output_chars=None):
    return __PRIVATE_TOOLS__.bash_exec(
        command=command,
        max_output_chars=max_output_chars,
    )

def fetch_webpage(url):
    return __PRIVATE_TOOLS__.fetch_webpage(url=url)

import json
import os
import sys

def _ghost_shell_quote(value):
    text = str(value)
    return "'" + text.replace("'", "'\"'\"'") + "'"

def _ghost_shell_command(args):
    if isinstance(args, str):
        return args
    if isinstance(args, (list, tuple)):
        return " ".join(_ghost_shell_quote(part) for part in args)
    raise TypeError("subprocess args must be a string or sequence")

def _ghost_shell_prefix(command, cwd=None, env=None):
    steps = []
    if cwd is not None:
        if not isinstance(cwd, str) or not cwd.strip():
            raise TypeError("cwd must be a non-empty string")
        steps.append("cd " + _ghost_shell_quote(cwd))
    if env is not None:
        items = getattr(env, "items", None)
        if not callable(items):
            raise TypeError("env must be a mapping")
        for key, value in items():
            normalized = key if isinstance(key, str) else ""
            if not normalized or not normalized.replace("_", "a").isalnum():
                raise ValueError("env keys must use letters, digits, or _")
            steps.append("export " + normalized + "=" + _ghost_shell_quote(value))
    steps.append(command)
    return " && ".join(steps)

def _ghost_shell_result(args, cwd=None, env=None, timeout=None):
    if timeout is None:
        timeout_ms = None
    else:
        if not isinstance(timeout, (int, float)) or timeout <= 0:
            raise ValueError("timeout must be a positive number")
        timeout_ms = int(timeout * 1000)
    command = _ghost_shell_prefix(_ghost_shell_command(args), cwd=cwd, env=env)
    payload = json.loads(
        __PRIVATE_TOOLS__._bash_exec_result(command=command, timeout_ms=timeout_ms)
    )
    status = payload.get("status") or ""
    if status.startswith("exit status: "):
        returncode = int(status.split(": ", 1)[1])
    else:
        returncode = 0 if payload.get("success") else 1
    return payload.get("stdout") or "", payload.get("stderr") or "", returncode

class _GhostCalledProcessError(RuntimeError):
    def __init__(self, returncode, cmd, output="", stderr=""):
        super().__init__(f"Command '{cmd}' returned non-zero exit status {returncode}.")
        self.returncode = returncode
        self.cmd = cmd
        self.output = output
        self.stderr = stderr

class _GhostCompletedProcess:
    def __init__(self, args, returncode, stdout="", stderr=""):
        self.args = args
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr

    def check_returncode(self):
        if self.returncode != 0:
            raise _GhostCalledProcessError(
                self.returncode,
                self.args,
                self.stdout,
                self.stderr,
            )

class _GhostPopenReader:
    def __init__(self, output):
        self._output = output

    def read(self):
        return self._output

    def readline(self):
        if not self._output:
            return ""
        line, _, remainder = self._output.partition("\n")
        self._output = remainder
        return line + ("\n" if remainder else "")

    def readlines(self):
        return self._output.splitlines(True)

    def close(self):
        self._output = ""

class _GhostSubprocessModule:
    PIPE = -1
    STDOUT = -2
    DEVNULL = -3
    CalledProcessError = _GhostCalledProcessError
    CompletedProcess = _GhostCompletedProcess

    def run(self, args, capture_output=False, text=True, check=False, shell=False, cwd=None, env=None, timeout=None, stdout=None, stderr=None):
        if stdout not in (None, self.PIPE):
            raise NotImplementedError("subprocess shim supports only stdout=None or PIPE")
        if stderr not in (None, self.PIPE, self.STDOUT):
            raise NotImplementedError("subprocess shim supports only stderr=None, PIPE, or STDOUT")
        out, err, returncode = _ghost_shell_result(args, cwd=cwd, env=env, timeout=timeout)
        if stderr == self.STDOUT:
            out, err = out + err, ""
        if not text:
            out, err = out.encode("utf-8"), err.encode("utf-8")
        completed = _GhostCompletedProcess(args, returncode, out, err)
        if check:
            completed.check_returncode()
        return completed

    def check_output(self, args, text=True, shell=False, cwd=None, env=None, timeout=None):
        return self.run(
            args,
            capture_output=True,
            text=text,
            check=True,
            shell=shell,
            cwd=cwd,
            env=env,
            timeout=timeout,
        ).stdout

def _ghost_os_popen(command, mode="r", buffering=-1):
    if mode not in ("r", "rt", "tr"):
        raise ValueError("os.popen shim only supports read mode")
    if buffering != -1:
        raise ValueError("os.popen shim only supports default buffering")
    stdout, _stderr, _returncode = _ghost_shell_result(command)
    return _GhostPopenReader(stdout)

def _ghost_os_system(command):
    _stdout, _stderr, returncode = _ghost_shell_result(command)
    return returncode

subprocess = _GhostSubprocessModule()
os.popen = _ghost_os_popen
os.system = _ghost_os_system
sys.modules["subprocess"] = subprocess
"#;

pub(super) fn build_helper_bootstrap(private_tools_name: &str) -> String {
    HELPER_BOOTSTRAP_TEMPLATE.replace("__PRIVATE_TOOLS__", private_tools_name)
}
