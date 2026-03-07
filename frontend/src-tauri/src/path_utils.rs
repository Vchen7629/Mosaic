use std::path::PathBuf;
use tauri::Manager;

pub fn get_backend_path(app_handle: &tauri::AppHandle) -> Result<PathBuf, String> {
    let resource_dir = app_handle
        .path()
        .resource_dir()
        .expect("failed to get resource dir");

    let is_dev_mode = resource_dir.to_string_lossy().contains("debug");

    if is_dev_mode {
        println!("[Rust] Development mode detected, using actual backend");
        let current_dir = std::env::current_dir().expect("failed to get current dir");
        Ok(current_dir
            .parent()
            .and_then(|p| p.parent())
            .expect("failed to determine workspace root")
            .join("backend"))
    } else {
        let backend_bundled_path = resource_dir.join("_up_").join("_up_").join("backend");
        let backend_direct_path = resource_dir.join("backend");

        if backend_bundled_path.exists() {
            println!("[Rust] Production - using bundled backend (_up_/_up_/backend)");
            Ok(backend_bundled_path)
        } else if backend_direct_path.exists() {
            println!("[Rust] Production - using bundled backend (direct)");
            Ok(backend_direct_path)
        } else {
            Err("Backend not found in production build".to_string())
        }
    }
}
