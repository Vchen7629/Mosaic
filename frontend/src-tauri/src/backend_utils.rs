use std::process::{Command, Stdio};
use tauri::State;
use crate::{BackendProcesses, port_utils, path_utils};

#[tauri::command]
pub fn start_backend_api(state: State<BackendProcesses>, app_handle: tauri::AppHandle) -> Result<String, String> {
    println!("[Rust] start_backend_api called");

    {
        let children = state.process_children.lock().unwrap();
        if !children.is_empty() {
            println!("[Rust] Backend already running with {} process(es)", children.len());
            return Ok("Backend already running".to_string());
        }
    }

    if port_utils::ports_in_use() {
        println!("[Rust] Port 8000 in use by untracked process, cleaning up...");
        port_utils::kill_processes_on_ports();
        port_utils::wait_for_ports_free(std::time::Duration::from_secs(5));
    }

    let backend_path = path_utils::get_backend_path(&app_handle)?;

    println!("[Rust] Backend resource path: {}", backend_path.display());

    #[cfg(target_os = "windows")]
    let venv_python = backend_path.join(".venv").join("Scripts").join("python.exe");
    #[cfg(not(target_os = "windows"))]
    let venv_python = backend_path.join(".venv").join("bin").join("python3");

    let venv_path = backend_path.join(".venv");
    #[cfg(target_os = "windows")]
    let venv_bin = venv_path.join("Scripts");
    #[cfg(not(target_os = "windows"))]
    let venv_bin = venv_path.join("bin");
    let current_path = std::env::var("PATH").unwrap_or_default();
    #[cfg(target_os = "windows")]
    let new_path = format!("{};{}", venv_bin.display(), current_path);
    #[cfg(not(target_os = "windows"))]
    let new_path = format!("{}:{}", venv_bin.display(), current_path);

    port_utils::wait_for_ports_free(std::time::Duration::from_secs(5));

    let workspace_root = backend_path
        .parent()
        .expect("failed to get workspace root from backend path");

    println!("[Rust] Workspace root: {}", workspace_root.display());
    println!("[Rust] Python executable: {}", venv_python.display());

    let mut backend_cmd = Command::new(&venv_python);
    backend_cmd
        .args(&["-m", "uvicorn", "src.main:app", "--host", "127.0.0.1", "--port", "8000"])
        .current_dir(&backend_path)
        .env("PATH", &new_path)
        .env("VIRTUAL_ENV", &venv_path)
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit());

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        const CREATE_NO_WINDOW: u32 = 0x08000000;
        backend_cmd.creation_flags(CREATE_NO_WINDOW);
    }

    let backend_process = backend_cmd
        .spawn()
        .map_err(|e| format!("Failed to start backend: {}\nWorkspace: {}\nPython: {}",
            e, workspace_root.display(), venv_python.display()))?;

    let backend_pid = backend_process.id();

    let mut children = state.process_children.lock().unwrap();
    children.clear();
    children.push(backend_process);

    println!("[Rust] Backend started successfully! PID: {}", backend_pid);

    port_utils::wait_for_port_ready(std::time::Duration::from_secs(15));

    Ok(format!("Backend started (PID: {})", backend_pid))
}

#[tauri::command]
pub fn stop_backend_api(state: State<BackendProcesses>) -> Result<String, String> {
    let mut guard = state.process_children.lock().unwrap();
    let mut children = Vec::new();
    std::mem::swap(&mut *guard, &mut children);
    drop(guard);

    let pids: Vec<u32> = children.iter().map(|c| c.id()).collect();
    println!("[Rust] stop_backend_api called. Stored PIDs: {:?}", pids);

    if pids.is_empty() && !port_utils::ports_in_use() {
        println!("[Rust] No tracked children and ports are clear; nothing to stop.");
        return Ok("Backend stopped".to_string());
    }

    for mut child in children {
        let pid = child.id();
        println!("[Rust] Attempting to kill child PID {} via Child::kill()", pid);
        match child.kill() {
            Ok(_) => {
                println!("[Rust] kill() succeeded for PID {}. Waiting for exit...", pid);
                match child.wait() {
                    Ok(status) => println!("[Rust] PID {} exited with status {}", pid, status),
                    Err(e) => println!("[Rust] waiting for PID {} failed: {}", pid, e),
                }
            }
            Err(e) => {
                println!("[Rust] Child::kill() failed for PID {}: {}", pid, e);
            }
        }
    }

    #[cfg(not(target_os = "windows"))]
    {
        for pid in &pids {
            println!("[Rust] Attempting pkill -P {} to kill any child processes", pid);
            let _ = Command::new("sh").args(["-c", &format!("pkill -P {} || true", pid)]).output();
        }
    }

    port_utils::kill_processes_on_ports();
    port_utils::verify_ports_clear_and_print();

    Ok("Backend stopped".to_string())
}
