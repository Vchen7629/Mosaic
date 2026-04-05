#[cfg(not(target_os = "windows"))]
use std::process::Command;
use tauri::State;
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandEvent;
use crate::{BackendProcesses, port_utils};

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

    {
        let mut is_starting = state.is_starting.lock().unwrap();
        if *is_starting {
            println!("[Rust] Backend already starting, ignoring duplicate call");
            return Ok("Backend already starting".to_string());
        }
        *is_starting = true;
    }

    if port_utils::ports_in_use() {
        println!("[Rust] Port 8080 in use by untracked process, cleaning up...");
        port_utils::kill_processes_on_ports();
        port_utils::wait_for_ports_free(std::time::Duration::from_secs(5));
    }

    let sidecar = app_handle
        .shell()
        .sidecar("backend-client")
        .map_err(|e| {
            *state.is_starting.lock().unwrap() = false;
            format!("Failed to create sidecar command: {}", e)
        })?;

    let (rx, child) = sidecar
        .spawn()
        .map_err(|e| {
            *state.is_starting.lock().unwrap() = false;
            format!("Failed to spawn backend sidecar: {}", e)
        })?;

    let pid = child.pid();
    println!("[Rust] Backend sidecar started successfully! PID: {}", pid);

    // Forward backend stdout/stderr so logs remain visible (mirrors the old Stdio::inherit() behaviour)
    let state_clone = state.inner().clone();
    tauri::async_runtime::spawn(async move {
        let mut rx = rx;
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => print!("[backend] {}", String::from_utf8_lossy(&line)),
                CommandEvent::Stderr(line) => eprint!("[backend] {}", String::from_utf8_lossy(&line)),
                CommandEvent::Terminated(status) => {
                    println!("[Rust] Backend process terminated: {:?}", status);
                    state_clone.process_children.lock().unwrap().clear();
                    break;
                }
                _ => {}
            }
        }
    });

    state.process_children.lock().unwrap().push(child);
    *state.is_starting.lock().unwrap() = false;

    Ok("Backend started".to_string())
}

#[tauri::command]
pub fn stop_backend_api(state: State<BackendProcesses>) -> Result<String, String> {
    let mut guard = state.process_children.lock().unwrap();
    let mut children = Vec::new();
    std::mem::swap(&mut *guard, &mut children);
    drop(guard);

    let pids: Vec<u32> = children.iter().map(|c| c.pid()).collect();
    println!("[Rust] stop_backend_api called. Stored PIDs: {:?}", pids);

    if pids.is_empty() && !port_utils::ports_in_use() {
        println!("[Rust] No tracked children and ports are clear; nothing to stop.");
        return Ok("Backend stopped".to_string());
    }

    for child in children {
        let pid = child.pid();
        println!("[Rust] Killing sidecar PID {}", pid);
        match child.kill() {
            Ok(_) => println!("[Rust] kill() succeeded for PID {}", pid),
            Err(e) => println!("[Rust] kill() failed for PID {}: {}", pid, e),
        }
    }

    #[cfg(not(target_os = "windows"))]
    {
        for pid in &pids {
            println!("[Rust] Attempting pkill -P {} to kill any child processes", pid);
            let _ = Command::new("sh").args(["-c", &format!("pkill -P {} || true", pid)]).output();
        }
    }

    // Wait for the process to fully exit and release the port before force-killing stragglers.
    // (CommandChild has no wait(), so we poll instead.)
    port_utils::wait_for_ports_free(std::time::Duration::from_secs(5));
    port_utils::kill_processes_on_ports();
    port_utils::verify_ports_clear_and_print();

    Ok("Backend stopped".to_string())
}
