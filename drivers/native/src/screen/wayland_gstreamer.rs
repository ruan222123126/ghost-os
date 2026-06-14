use super::image_ops::load_image_from_path;
use std::io;
use std::os::fd::RawFd;
use std::os::unix::process::CommandExt;
use std::path::{Path, PathBuf};
use std::process::{Child, Command, ExitStatus, Stdio};
use std::thread;
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use xcap::image::RgbaImage;

const GST_CAPTURE_TIMEOUT: Duration = Duration::from_secs(8);
const PIPEWIRE_FD_IN_CHILD: RawFd = 3;

pub(crate) fn capture_png_from_pipewire(
    source_fd: RawFd,
    node_id: u32,
) -> Result<RgbaImage, String> {
    let path = unique_temp_png_path();
    run_gstreamer_capture(source_fd, node_id, &path)?;
    let image = load_image_from_path(path.to_string_lossy().as_ref())
        .map_err(|err| format!("load gstreamer capture image failed: {err}"))?;
    let _ = std::fs::remove_file(&path);
    Ok(image)
}

fn run_gstreamer_capture(source_fd: RawFd, node_id: u32, output_path: &Path) -> Result<(), String> {
    let mut command = Command::new("gst-launch-1.0");
    command
        .arg("-q")
        .arg(format!(
            "pipewiresrc fd={PIPEWIRE_FD_IN_CHILD} path={node_id} do-timestamp=true"
        ))
        .arg("!")
        .arg("videoconvert")
        .arg("!")
        .arg("video/x-raw,format=RGBA")
        .arg("!")
        .arg("pngenc")
        .arg("snapshot=true")
        .arg("!")
        .arg("filesink")
        .arg(format!("location={}", output_path.to_string_lossy()))
        .stdout(Stdio::null())
        .stderr(Stdio::null());
    attach_pipewire_fd(&mut command, source_fd);
    let mut child = command
        .spawn()
        .map_err(|err| format!("spawn gst-launch-1.0 failed: {err}"))?;
    let status = wait_child_with_timeout(&mut child, GST_CAPTURE_TIMEOUT)?;
    if status.success() {
        return Ok(());
    }
    Err(format!("gst-launch-1.0 exited with status {status}"))
}

fn attach_pipewire_fd(command: &mut Command, source_fd: RawFd) {
    // SAFETY: pre_exec is required to duplicate the portal fd into the child process.
    unsafe {
        command.pre_exec(move || duplicate_fd_for_child(source_fd, PIPEWIRE_FD_IN_CHILD));
    }
}

fn duplicate_fd_for_child(source_fd: RawFd, target_fd: RawFd) -> io::Result<()> {
    if source_fd == target_fd {
        return clear_cloexec(target_fd);
    }
    if unsafe { libc::dup2(source_fd, target_fd) } == -1 {
        return Err(io::Error::last_os_error());
    }
    clear_cloexec(target_fd)
}

fn clear_cloexec(fd: RawFd) -> io::Result<()> {
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFD) };
    if flags == -1 {
        return Err(io::Error::last_os_error());
    }
    if unsafe { libc::fcntl(fd, libc::F_SETFD, flags & !libc::FD_CLOEXEC) } == -1 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

fn wait_child_with_timeout(child: &mut Child, timeout: Duration) -> Result<ExitStatus, String> {
    let deadline = Instant::now() + timeout;
    loop {
        if let Some(status) = child
            .try_wait()
            .map_err(|err| format!("wait gst-launch-1.0 failed: {err}"))?
        {
            return Ok(status);
        }
        if Instant::now() >= deadline {
            let _ = child.kill();
            let _ = child.wait();
            return Err(format!("gst-launch-1.0 timed out after {timeout:?}"));
        }
        thread::sleep(Duration::from_millis(50));
    }
}

fn unique_temp_png_path() -> PathBuf {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|value| value.as_nanos())
        .unwrap_or(0);
    std::env::temp_dir().join(format!("ghost-os-wayland-screencast-{nanos}.png"))
}
