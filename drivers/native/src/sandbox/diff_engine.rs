// Unified diff parser and patch applier used by sandbox file tools.

use regex::Regex;
use std::sync::OnceLock;

pub(crate) struct PatchResult {
    pub(crate) updated: String,
    pub(crate) hunk_count: usize,
}

pub(crate) fn apply_unified_patch(original: &str, diff_text: &str) -> Result<PatchResult, String> {
    let hunks = parse_unified_diff(diff_text)?;
    let updated = apply_unified_diff(original, &hunks)?;
    Ok(PatchResult {
        updated,
        hunk_count: hunks.len(),
    })
}

#[derive(Debug, Clone)]
struct DiffHunk {
    old_start: usize,
    lines: Vec<DiffLine>,
}

#[derive(Debug, Clone)]
enum DiffLine {
    Context(String),
    Add(String),
    Remove(String),
}

fn hunk_header_regex() -> &'static Regex {
    static HUNK_RE: OnceLock<Regex> = OnceLock::new();
    HUNK_RE.get_or_init(|| {
        Regex::new(r"^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@").expect("valid hunk regex")
    })
}

fn parse_unified_diff(diff_text: &str) -> Result<Vec<DiffHunk>, String> {
    let mut lines = diff_text.lines().peekable();
    let mut hunks = Vec::new();
    let hunk_re = hunk_header_regex();

    while let Some(raw_line) = lines.next() {
        let line = raw_line.strip_suffix('\r').unwrap_or(raw_line);
        let captures = match hunk_re.captures(line) {
            Some(captures) => captures,
            None => continue,
        };

        let old_start = captures
            .get(1)
            .and_then(|value| value.as_str().parse::<usize>().ok())
            .ok_or_else(|| format!("invalid hunk header: {line}"))?;

        let mut hunk_lines = Vec::new();
        while let Some(next) = lines.peek() {
            let candidate = next.strip_suffix('\r').unwrap_or(next);
            if hunk_re.is_match(candidate) {
                break;
            }

            let next = lines.next().unwrap_or_default();
            let candidate = next.strip_suffix('\r').unwrap_or(next);
            if candidate == "\\ No newline at end of file" {
                continue;
            }

            let mut chars = candidate.chars();
            let marker = chars
                .next()
                .ok_or_else(|| "malformed diff line".to_string())?;
            let value: String = chars.collect();

            match marker {
                ' ' => hunk_lines.push(DiffLine::Context(value)),
                '+' => hunk_lines.push(DiffLine::Add(value)),
                '-' => hunk_lines.push(DiffLine::Remove(value)),
                _ => return Err(format!("malformed diff line: {candidate}")),
            }
        }

        if hunk_lines.is_empty() {
            return Err("diff hunk has no body".to_string());
        }

        hunks.push(DiffHunk {
            old_start,
            lines: hunk_lines,
        });
    }

    if hunks.is_empty() {
        return Err("no diff hunks found".to_string());
    }

    Ok(hunks)
}

fn apply_unified_diff(original: &str, hunks: &[DiffHunk]) -> Result<String, String> {
    let source_lines: Vec<&str> = original.lines().collect();
    let had_trailing_newline = original.ends_with('\n');

    let mut result = Vec::new();
    let mut source_index = 0usize;

    for hunk in hunks {
        let hunk_start = hunk.old_start.saturating_sub(1);
        if hunk_start < source_index {
            return Err("invalid diff order: overlapping hunks".to_string());
        }
        if hunk_start > source_lines.len() {
            return Err(format!(
                "hunk start {} exceeds file length {}",
                hunk.old_start,
                source_lines.len()
            ));
        }

        for line in &source_lines[source_index..hunk_start] {
            result.push((*line).to_string());
        }

        let mut cursor = hunk_start;
        for line in &hunk.lines {
            match line {
                DiffLine::Context(expected) => {
                    let actual = source_lines
                        .get(cursor)
                        .ok_or_else(|| "diff context exceeds file length".to_string())?;
                    if actual != &expected.as_str() {
                        return Err(format!(
                            "diff context mismatch at line {}",
                            cursor.saturating_add(1)
                        ));
                    }
                    result.push(expected.clone());
                    cursor = cursor.saturating_add(1);
                }
                DiffLine::Remove(expected) => {
                    let actual = source_lines
                        .get(cursor)
                        .ok_or_else(|| "diff removal exceeds file length".to_string())?;
                    if actual != &expected.as_str() {
                        return Err(format!(
                            "diff removal mismatch at line {}",
                            cursor.saturating_add(1)
                        ));
                    }
                    cursor = cursor.saturating_add(1);
                }
                DiffLine::Add(value) => {
                    result.push(value.clone());
                }
            }
        }

        source_index = cursor;
    }

    for line in &source_lines[source_index..] {
        result.push((*line).to_string());
    }

    let mut patched = result.join("\n");
    if had_trailing_newline {
        patched.push('\n');
    }
    Ok(patched)
}
