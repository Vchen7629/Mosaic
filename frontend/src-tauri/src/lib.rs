use std::sync::Mutex;
mod backend_utils;
mod port_utils;
mod path_utils;

struct BackendProcesses {
    process_children: Mutex<Vec<std::process::Child>>,
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(BackendProcesses {
            process_children: Mutex::new(Vec::new()),
        })
        .invoke_handler(tauri::generate_handler![
            backend_utils::start_backend_api,
            backend_utils::stop_backend_api,
        ])
        .plugin(tauri_plugin_opener::init())
        .setup(|_app| {
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
