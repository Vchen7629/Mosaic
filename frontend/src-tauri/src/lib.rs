use std::sync::{Arc, Mutex};
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
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
