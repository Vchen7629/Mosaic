use std::process::{Command, Stdio};
use tauri::State;
use crate::{CaptionProcesses, port_utils, path_utils};

#[tauri::command]
pub fn start_backend_api(state: State<CaptionProcesses>, app_handle: tauri::AppHandle) -> Result<String, String> {
    // Minimal startup: stop any tracked processes, then start services.
    println!("[Rust] start_backend_api called");

    // Check if backend already running
    {
        let children = state.process_children.lock().unwrap();
        if !children.is_empty() {
            println!("[Rust] Backend already running with {} process(es)", children.len());
            return Ok("Backend already running".to_string());
        }
    }

    // If port is in use but we dont have a tracked process, clean it up
    if port_utils::ports_in_use() {
        println!("[Rust] Port 8000 in use by untracked process, cleaning up...");
        port_utils::kill_processes_on_ports();
        port_utils::wait_for_ports_free(std::time::Duration::from_secs(5));
    }

    let backend_path_candidate = path_utils::get_backend_path(&app_handle)?;

    println!("[Rust] Backend resource path: {}", backend_path_candidate.display());

    let backend_path = backend_path_candidate.join("src").join("translation");

    #[cfg(target_os = "windows")]
    let venv_python = backend_path_candidate.join(".venv").join("Scripts").join("python.exe");
    #[cfg(not(target_os = "windows"))]
    let venv_python = backend_path_candidate.join(".venv").join("bin").join("python3");

    let venv_path = backend_path_candidate.join(".venv");
    #[cfg(target_os = "windows")]
    let venv_bin = venv_path.join("Scripts");
    #[cfg(not(target_os = "windows"))]
    let venv_bin = venv_path.join("bin");
    let current_path = std::env::var("PATH").unwrap_or_default();
    #[cfg(target_os = "windows")]
    let new_path = format!("{};{}", venv_bin.display(), current_path);
    #[cfg(not(target_os = "windows"))]
    let new_path = format!("{}:{}", venv_bin.display(), current_path);

    // Wait a short time for any previous services to fully exit before
    // attempting start. This is a defensive wait, not an unconditional kill.
    port_utils::wait_for_ports_free(std::time::Duration::from_secs(5));

    // Get the parent directory of the backend (the workspace root for the bundled app)
    let workspace_root = backend_path_candidate
        .parent()
        .expect("failed to get workspace root from backend path");

    println!("[Rust] Workspace root: {}", workspace_root.display());
    println!("[Rust] Python executable: {}", venv_python.display());
    println!("[Rust] Venv path: {}", venv_path.display());

    // Get app data directory for production paths
    let matrix_config_path = path_utils::get_matrix_config_path(&app_handle);
    let db_path = path_utils::get_db_path(&app_handle);
    let log_dir = path_utils::get_log_dir(&app_handle);

    // Start backend service (FastAPI server) using uvicorn (runs both websocket and apis)
    let mut backend_cmd = Command::new(&venv_python);
    backend_cmd
        .args(&["-m", "uvicorn", "backend.src.main:app", "--host", "127.0.0.1", "--port", "8000"])
        .current_dir(&workspace_root)
        .env("PATH", &new_path)
        .env("VIRTUAL_ENV", &venv_path)
        .env("PYTHONPATH", &backend_path)
        .env("MATRIX_CONFIG_PATH", matrix_config_path.to_str().unwrap())
        .env("DB_PATH", db_path.to_str().unwrap())
        .env("LOG_DIR", log_dir.to_str().unwrap())
        .stdout(Stdio::null())
        .stderr(Stdio::null());

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

    // Store the Child handles so we can call .kill()/.wait() later.
    let mut children = state.process_children.lock().unwrap();
    children.clear();
    children.push(backend_process);

    println!("[Rust] All services started successfully!");
    println!("[Rust] PIDs: backend={}", backend_pid);
    
    Ok(format!("Caption services started (PIDs: {})", backend_pid))
}

#[tauri::command]
pub fn stop_backend_api(state: State<CaptionProcesses>) -> Result<String, String> {
    // Take ownership of the Child handles so we can kill/wait them reliably.
    let mut guard = state.process_children.lock().unwrap();
    let mut children = Vec::new();
    std::mem::swap(&mut *guard, &mut children);
    drop(guard);

    let pids: Vec<u32> = children.iter().map(|c| c.id()).collect();
    println!("[Rust] stop_transcription_services called. Stored PIDs: {:?}", pids);

    // If there are no tracked children and ports are clear, exit early.
    if pids.is_empty() && !port_utils::ports_in_use() {
        println!("[Rust] No tracked children and ports are clear; nothing to stop.");
        return Ok("Caption services stopped".to_string());
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
                // Fallbacks below will attempt OS-level kills
            }
        }
    }

    // On Unix-like systems, attempt to kill any descendant processes of the
    // PIDs we tracked (handles cases where children spawned grandchildren).
    #[cfg(not(target_os = "windows"))]
    {
        for pid in &pids {
            println!("[Rust] Attempting pkill -P {} to kill any child processes", pid);
            let _ = Command::new("sh").args(["-c", &format!("pkill -P {} || true", pid)]).output();
        }
    }

    // Final fallback: detect any remaining processes listening on the standard
    // backend/websocket ports and kill them. This handles processes that were
    // started outside this Tauri run or re-parented and escaped the stored Childs.
    // Final fallback: inspect ports and kill any remaining processes, then
    // verify and print success if ports are clear.
    port_utils::kill_processes_on_ports();
    port_utils::verify_ports_clear_and_print();

    Ok("Caption services stopped".to_string())
}