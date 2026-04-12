from __future__ import annotations

import re
from bisect import bisect_right
from dataclasses import dataclass
from pathlib import Path

MAX_FILE_LINES = 300
MAX_FUNCTION_LINES = 50
MAX_NESTING = 3
MAX_PARAMS = 3
MAX_COMPLEXITY = 10

GO_FUNC_RE = re.compile(r"(?m)^\s*func\s+(?:\([^\n)]*\)\s*)?[A-Za-z_][A-Za-z0-9_]*(?:\s*\[[^\]]+\])?\s*\(")
RUST_FUNC_RE = re.compile(r"(?m)^\s*(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?fn\s+[A-Za-z_][A-Za-z0-9_]*\s*(?:<[^>{}]*>\s*)?\(")
JS_FUNC_RE = re.compile(r"(?m)^\s*(?:export\s+)?(?:async\s+)?function\s+[A-Za-z_$][A-Za-z0-9_$]*\s*\(")
JS_ARROW_RE = re.compile(r"(?m)^\s*(?:export\s+)?(?:const|let|var)\s+[A-Za-z_$][A-Za-z0-9_$]*\s*=\s*(?:async\s*)?\(")
CONTROL_START_RE = re.compile(r"\b(if|for|while|switch|match|catch|else)\b")
WEB_IMPORT_RE = re.compile(r"(?:from\s+['\"]([^'\"]+)['\"]|import\s+['\"]([^'\"]+)['\"]|require\(['\"]([^'\"]+)['\"]\))")
COMPLEXITY_PATTERNS = (
    re.compile(r"\bif\b"),
    re.compile(r"\bfor\b"),
    re.compile(r"\bwhile\b"),
    re.compile(r"\bcase\b"),
    re.compile(r"\bcatch\b"),
    re.compile(r"\bmatch\b"),
    re.compile(r"&&"),
    re.compile(r"\|\|"),
    re.compile(r"\?"),
    re.compile(r"=>"),
)
@dataclass(frozen=True)
class FunctionMetric:
    name: str
    start_line: int
    lines: int
    params: int
    complexity: int
    nesting: int
@dataclass(frozen=True)
class SourceView:
    clean: str
    text: str
    lines: list[str]
    starts: list[int]
    suffix: str
def line_starts(text: str) -> list[int]:
    starts = [0]
    for idx, char in enumerate(text):
        if char == "\n":
            starts.append(idx + 1)
    return starts
def line_number(starts: list[int], position: int) -> int:
    return bisect_right(starts, position)

def mask(char: str) -> str:
    return "\n" if char == "\n" else " "
def consume_active_state(state: str, text: str, index: int) -> tuple[str, int, str]:
    char = text[index]
    pair = text[index : index + 2]
    if state == "line_comment":
        return ("code" if char == "\n" else state), index + 1, mask(char)
    if state == "block_comment":
        if pair == "*/":
            return "code", index + 2, "  "
        return state, index + 1, mask(char)
    if state == "double_quote":
        if char == "\\":
            return state, index + 2, "  "
        return ("code" if char == '"' else state), index + 1, mask(char)
    return ("code" if char == "`" else state), index + 1, mask(char)
def strip_non_code(text: str) -> str:
    out: list[str] = []
    i = 0
    state = "code"
    while i < len(text):
        char = text[i]
        pair = text[i : i + 2]
        if state != "code":
            state, i, chunk = consume_active_state(state, text, i)
            out.extend(chunk)
            continue
        if pair == "//":
            out.extend("  ")
            state = "line_comment"
            i += 2
            continue
        if pair == "/*":
            out.extend("  ")
            state = "block_comment"
            i += 2
            continue
        if char == '"':
            out.append(" ")
            state = "double_quote"
            i += 1
            continue
        if char == "`":
            out.append(" ")
            state = "backtick"
            i += 1
            continue
        out.append(char)
        i += 1
    return "".join(out)

def find_matching_paren(clean: str, start: int) -> int:
    depth = 0
    for idx in range(start, len(clean)):
        char = clean[idx]
        if char == "(":
            depth += 1
        if char == ")":
            depth -= 1
            if depth == 0:
                return idx
    return -1

def find_matching_brace(clean: str, start: int) -> int:
    depth = 0
    for idx in range(start, len(clean)):
        char = clean[idx]
        if char == "{":
            depth += 1
        if char == "}":
            depth -= 1
            if depth == 0:
                return idx
    return -1

def split_top_level(expr: str) -> list[str]:
    items: list[str] = []
    current: list[str] = []
    round_depth = square_depth = curly_depth = angle_depth = 0
    for char in expr:
        if char == "(":
            round_depth += 1
        elif char == ")" and round_depth > 0:
            round_depth -= 1
        elif char == "[":
            square_depth += 1
        elif char == "]" and square_depth > 0:
            square_depth -= 1
        elif char == "{":
            curly_depth += 1
        elif char == "}" and curly_depth > 0:
            curly_depth -= 1
        elif char == "<":
            angle_depth += 1
        elif char == ">" and angle_depth > 0:
            angle_depth -= 1

        if char == "," and not any((round_depth, square_depth, curly_depth, angle_depth)):
            token = "".join(current).strip()
            if token:
                items.append(token)
            current = []
            continue
        current.append(char)

    token = "".join(current).strip()
    if token:
        items.append(token)
    return items

def function_patterns(suffix: str) -> list[re.Pattern[str]]:
    if suffix == ".go":
        return [GO_FUNC_RE]
    if suffix == ".rs":
        return [RUST_FUNC_RE]
    return [JS_FUNC_RE, JS_ARROW_RE]

def extract_name(signature: str, suffix: str) -> str:
    if suffix == ".go":
        match = re.search(r"func\s+(?:\([^\n)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)", signature)
        return match.group(1) if match else "<anonymous>"
    if suffix == ".rs":
        match = re.search(r"fn\s+([A-Za-z_][A-Za-z0-9_]*)", signature)
        return match.group(1) if match else "<anonymous>"
    named = re.search(r"function\s+([A-Za-z_$][A-Za-z0-9_$]*)", signature)
    if named:
        return named.group(1)
    assigned = re.search(r"(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)", signature)
    return assigned.group(1) if assigned else "<anonymous>"

def cyclomatic_complexity(body: str) -> int:
    score = 1
    for pattern in COMPLEXITY_PATTERNS:
        score += len(pattern.findall(body))
    return score

def max_nesting(body: str) -> int:
    pending_control = False
    control_depth = 0
    max_depth = 0
    brace_marks: list[bool] = []
    idx = 0
    while idx < len(body):
        match = CONTROL_START_RE.match(body, idx)
        if match:
            pending_control = True
            idx = match.end()
            continue
        char = body[idx]
        if char == "{":
            brace_marks.append(pending_control)
            if pending_control:
                control_depth += 1
                max_depth = max(max_depth, control_depth)
            pending_control = False
        elif char == "}":
            if brace_marks and brace_marks.pop() and control_depth > 0:
                control_depth -= 1
            pending_control = False
        elif char in ("\n", ";"):
            pending_control = False
        idx += 1
    return max_depth

def analyze_functions(path: Path, text: str) -> list[FunctionMetric]:
    view = SourceView(
        clean=strip_non_code(text),
        text=text,
        lines=text.splitlines(),
        starts=line_starts(text),
        suffix=path.suffix,
    )
    metrics: dict[tuple[str, int], FunctionMetric] = {}
    for pattern in function_patterns(view.suffix):
        for match in pattern.finditer(view.clean):
            metric = build_metric(match.start(), view)
            if metric:
                metrics[(metric.name, metric.start_line)] = metric
    return [metrics[key] for key in sorted(metrics)]

def build_metric(match_start: int, view: SourceView) -> FunctionMetric | None:
    param_start = view.clean.find("(", match_start)
    param_end = find_matching_paren(view.clean, param_start) if param_start >= 0 else -1
    body_start = view.clean.find("{", param_end + 1) if param_end >= 0 else -1
    body_end = find_matching_brace(view.clean, body_start) if body_start >= 0 else -1
    if min(param_start, param_end, body_start, body_end) < 0:
        return None
    start_line = line_number(view.starts, match_start)
    end_line = line_number(view.starts, body_end)
    non_blank = sum(1 for line in view.lines[start_line - 1 : end_line] if line.strip())
    signature = view.clean[match_start:body_start]
    body = view.clean[body_start + 1 : body_end]
    return FunctionMetric(
        name=extract_name(signature, view.suffix),
        start_line=start_line,
        lines=non_blank,
        params=len(split_top_level(view.text[param_start + 1 : param_end])),
        complexity=cyclomatic_complexity(body),
        nesting=max_nesting(body),
    )

def boundary_violations(path: Path, text: str) -> list[str]:
    rel = path.as_posix()
    violations: list[str] = []
    if rel.startswith("core/bridge/"):
        for imp in re.findall(r'"([^"]+)"', text):
            if imp.startswith("ghost-os/drivers") or imp.startswith("ghost-os/apps"):
                violations.append(f"{path}: forbidden cross-layer import '{imp}'")
    if rel.startswith("apps/web/"):
        imports = [item for group in WEB_IMPORT_RE.findall(text) for item in group if item]
        for imp in imports:
            if "core/bridge" in imp or "drivers/native" in imp:
                violations.append(f"{path}: forbidden cross-layer import '{imp}'")
    if rel.startswith("drivers/native/") and ("../core/bridge" in text or "../apps/web" in text):
        violations.append(f"{path}: forbidden cross-layer path reference")
    return violations

def analyze_targets(paths: list[Path]) -> list[str]:
    violations: list[str] = []
    for path in paths:
        text = path.read_text(encoding="utf-8")
        if len(text.splitlines()) > MAX_FILE_LINES:
            violations.append(f"{path}: file has more than {MAX_FILE_LINES} lines")
        violations.extend(boundary_violations(path, text))
        violations.extend(function_violations(path, text))
    return violations

def function_violations(path: Path, text: str) -> list[str]:
    violations: list[str] = []
    for metric in analyze_functions(path, text):
        if metric.lines > MAX_FUNCTION_LINES:
            violations.append(f"{path}:{metric.start_line}: '{metric.name}' lines={metric.lines} limit={MAX_FUNCTION_LINES}")
        if metric.params > MAX_PARAMS:
            violations.append(f"{path}:{metric.start_line}: '{metric.name}' params={metric.params} limit={MAX_PARAMS}")
        if metric.nesting > MAX_NESTING:
            violations.append(f"{path}:{metric.start_line}: '{metric.name}' nesting={metric.nesting} limit={MAX_NESTING}")
        if metric.complexity > MAX_COMPLEXITY:
            violations.append(
                f"{path}:{metric.start_line}: '{metric.name}' complexity={metric.complexity} limit={MAX_COMPLEXITY}"
            )
    return violations
