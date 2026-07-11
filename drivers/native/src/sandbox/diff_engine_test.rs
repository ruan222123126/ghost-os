use super::diff_engine::apply_unified_patch;

#[test]
fn apply_unified_patch_supports_multiple_hunks() {
    let original = "a\nb\nc\nd\n";
    let diff = "\
@@ -1,2 +1,2 @@
 a
-b
+b2
@@ -3,2 +3,2 @@
 c
-d
+d2
";
    let result = apply_unified_patch(original, diff).expect("patch should apply");
    assert_eq!(result.updated, "a\nb2\nc\nd2\n");
    assert_eq!(result.hunk_count, 2);
    assert_eq!(result.added_lines, 2);
    assert_eq!(result.removed_lines, 2);
    assert_eq!(result.hunk_ranges.len(), 2);
}

#[test]
fn apply_unified_patch_rejects_overlapping_hunks() {
    let original = "a\nb\nc\n";
    let diff = "\
@@ -2,1 +2,1 @@
-b
+B
@@ -2,1 +2,1 @@
-b
+B
";
    let result = apply_unified_patch(original, diff);
    assert!(result.is_err(), "overlap must fail");
    let err = result.err().unwrap_or_default();
    assert!(err.contains("invalid diff order"));
}

#[test]
fn apply_unified_patch_rejects_context_mismatch() {
    let original = "one\ntwo\nthree\n";
    let diff = "\
@@ -1,3 +1,3 @@
 one
-xxx
+TWO
 three
";
    let result = apply_unified_patch(original, diff);
    assert!(result.is_err(), "mismatch must fail");
    let err = result.err().unwrap_or_default();
    assert!(err.contains("mismatch"));
}

#[test]
fn apply_unified_patch_rejects_hunk_beyond_file_length() {
    let original = "line\n";
    let diff = "\
@@ -3,1 +3,1 @@
-line
+line2
";
    let result = apply_unified_patch(original, diff);
    assert!(result.is_err(), "out of range must fail");
    let err = result.err().unwrap_or_default();
    assert!(err.contains("exceeds file length"));
}

#[test]
fn apply_unified_patch_rejects_empty_hunk_body() {
    let original = "line\n";
    let diff = "@@ -1,1 +1,1 @@\n";
    let result = apply_unified_patch(original, diff);
    assert!(result.is_err(), "empty hunk must fail");
    let err = result.err().unwrap_or_default();
    assert!(err.contains("no body"));
}

#[test]
fn apply_unified_patch_rejects_malformed_diff_line() {
    let original = "line\n";
    let diff = "\
@@ -1,1 +1,1 @@
?line
";
    let result = apply_unified_patch(original, diff);
    assert!(result.is_err(), "malformed line must fail");
    let err = result.err().unwrap_or_default();
    assert!(err.contains("malformed diff line"));
}

#[test]
fn apply_unified_patch_supports_no_newline_marker() {
    let original = "alpha\nbeta\ngamma";
    let diff = "\
@@ -2,2 +2,2 @@
-beta
+beta2
 gamma
\\ No newline at end of file
";
    let result = apply_unified_patch(original, diff).expect("patch should apply");
    assert_eq!(result.updated, "alpha\nbeta2\ngamma");
}

#[test]
fn apply_unified_patch_supports_crlf_diff_lines() {
    let original = "hello\nworld\n";
    let diff = "@@ -1,2 +1,2 @@\r\n hello\r\n-world\r\n+rust\r\n";
    let result = apply_unified_patch(original, diff).expect("patch should apply");
    assert_eq!(result.updated, "hello\nrust\n");
}
