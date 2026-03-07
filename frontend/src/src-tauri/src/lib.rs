use tauri::Manager;
use std::sync::Mutex;

mod obs_utils;
mod backend_utils;
mod setup_utils;
mod port_utils;
mod path_utils;

struct CaptionProcesses {
    // Store the actual Child handles so we can reliably kill exactly the
    // processes we spawned (safer than numeric PIDs which can be reused).
    process_children: Mutex<Vec<std::process::Child>>,
    
}


#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
  tauri::Builder::default()
    .manage(CaptionProcesses {
        process_children: Mutex::new(Vec::new()),
    })
    .invoke_handler(tauri::generate_handler![
        setup_utils::check_setup_status,
        setup_utils::run_first_time_setup,
        setup_utils::mark_setup_complete,
        backend_utils::start_backend_api,
        backend_utils::stop_backend_api,
        obs_utils::check_file_exists,
        obs_utils::is_obs_process_running,
        obs_utils::launch_application,
        obs_utils::quit_obs_gracefully
    ])
    .plugin(tauri_plugin_shell::init())
    .plugin(tauri_plugin_http::init())
    .plugin(tauri_plugin_store::Builder::new().build())
    .setup(|app| {
        // Check if setup is complete
        let setup_complete = setup_utils::is_setup_complete(app.handle());

        if setup_complete {
            // Setup already done, show main window
            println!("[Rust] Setup already complete, showing main window");
            if let Some(main_window) = app.get_webview_window("main") {
                let _ = main_window.show();
            }
            // Hide setup window if it exists
            if let Some(setup_window) = app.get_webview_window("setup") {
                let _ = setup_window.hide();
            }
        } else {
            // Setup needed, show setup window and hide main window
            println!("[Rust] Setup needed, showing setup window");
            if let Some(setup_window) = app.get_webview_window("setup") {
                let _ = setup_window.show();
            }
            if let Some(main_window) = app.get_webview_window("main") {
                let _ = main_window.hide();
            }
        }

        Ok(())
    })
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}