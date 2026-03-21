use std::process::{Command, Stdio};
use tauri::State;
use crate::{BackendProcesses, port_utils, path_utils};

#[tauri::command]
pub fn start_backend_api(state: State<BackendProcesses>, app_handle: tauri::AppHandle) -> Result<String, String> {
    println!("[Rust] start_backend_api called");

    // Check if already running
    {
        let children = state.process_children.lock().unwrap();
        if !children.is_empty() {
            println!("[Rust] Backend already running with {} process(es)", children.len());
            return Ok("Backend already running".to_string());
        }
    }

    {
        let mut is_starting = state.is_starting.lock().unwrap();
        if *is_starting {
            println!("[Rust] Backend already starting, ignoring duplicate call");
            return Ok("Backend already starting".to_string());
        }
        *is_starting = true;
    }

    // Clone state for background thread
    let state_clone = state.inner().clone();

    // Spawn backend startup in background thread to avoid blocking UI
    std::thread::spawn(move || {
        println!("[Rust] Background thread: Starting backend startup sequence");

        // Check and clean up ports if needed
        if port_utils::ports_in_use() {
            println!("[Rust] Port 8080 in use by untracked process, cleaning up...");
            port_utils::kill_processes_on_ports();
            port_utils::wait_for_ports_free(std::time::Duration::from_secs(5));
        }

        let backend_path = match path_utils::get_backend_path(&app_handle) {
            Ok(p) => p,
            Err(e) => {
                eprintln!("[Rust] Failed to get backend path: {}", e);
                *state_clone.is_starting.lock().unwrap() = false;
                return;
            }
        };

        println!("[Rust] Backend resource path: {}", backend_path.display());

        // Go client is in backend/client directory
        let go_client_path = backend_path.join("client");

        if !go_client_path.exists() {
            eprintln!("[Rust] Go client directory not found at: {}", go_client_path.display());
            *state_clone.is_starting.lock().unwrap() = false;
            return;
        }

        let workspace_root = backend_path
            .parent()
            .expect("failed to get workspace root from backend path");

        println!("[Rust] Workspace root: {}", workspace_root.display());
        println!("[Rust] Go client path: {}", go_client_path.display());

        let mut backend_cmd = Command::new("go");
        backend_cmd
            .args(&["run", "./cmd/main.go"])
            .current_dir(&go_client_path)
            .stdout(Stdio::inherit())
            .stderr(Stdio::inherit());

        let backend_process = match backend_cmd.spawn() {
            Ok(p) => p,
            Err(e) => {
                eprintln!("[Rust] Failed to start Go backend: {}\nWorkspace: {}\nGo client path: {}",
                    e, workspace_root.display(), go_client_path.display());
                *state_clone.is_starting.lock().unwrap() = false;
                return;
            }
        };

        let backend_pid = backend_process.id();

        // Store the process
        let mut children = state_clone.process_children.lock().unwrap();
        children.clear();
        children.push(backend_process);

        println!("[Rust] Backend started successfully! PID: {}", backend_pid);

        // Reset the starting flag now that we've successfully started
        *state_clone.is_starting.lock().unwrap() = false;
    });

    // Return immediately to UI
    Ok("Backend starting in background...".to_string())
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
