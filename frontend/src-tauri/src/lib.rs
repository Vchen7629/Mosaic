use std::sync::{Arc, Mutex};
use tauri::Manager;
use tauri_plugin_shell::process::CommandChild;
mod backend_utils;
mod port_utils;

#[derive(Clone)]
struct BackendProcesses {
    process_children: Arc<Mutex<Vec<CommandChild>>>,
    is_starting: Arc<Mutex<bool>>,
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(BackendProcesses {
            process_children: Arc::new(Mutex::new(Vec::new())),
            is_starting: Arc::new(Mutex::new(false)),
        })
        .invoke_handler(tauri::generate_handler![
            backend_utils::start_backend_api,
            backend_utils::stop_backend_api,
        ])
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_shell::init())
        .setup(|_app| {
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|app, event| {
            if let tauri::RunEvent::ExitRequested { .. } = event {
                let state = app.state::<BackendProcesses>();
                let mut children = state.process_children.lock().unwrap();
                for child in children.drain(..) {
                    let pid = child.pid();
                    if let Err(e) = child.kill() {
                        eprintln!("[Rust] Failed to kill backend PID {} on exit: {}", pid, e);
                    } else {
                        println!("[Rust] Killed backend PID {} on app exit", pid);
                    }
                }
                port_utils::kill_processes_on_ports();
            }
        });
}
