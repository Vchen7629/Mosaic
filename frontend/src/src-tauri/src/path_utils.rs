use std::path::PathBuf;
use tauri::Manager;

pub fn get_backend_path(app_handle: &tauri::AppHandle) -> Result<PathBuf, String> {
    // Determine backend path based on whether we're in dev or production
    // In dev: backend is at ../../backend (from src-tauri directory)
    // In production: backend is bundled in the resource directory as _up_/_up_/backend
    let resource_dir = app_handle
        .path()
        .resource_dir()
        .expect("failed to get resource dir");

    // Check if we're in dev mode (debug build)
    let is_dev_mode = resource_dir.to_string_lossy().contains("debug");

    if is_dev_mode {
        // Development: use actual backend directory
        println!("[Rust] Development mode detected, using actual backend");
        let current_dir = std::env::current_dir().expect("failed to get current dir");
        Ok(current_dir
            .parent()
            .and_then(|p| p.parent())
            .expect("failed to determine workspace root")
            .join("backend"))
    } else {
        // Production: use bundled backend
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

pub fn get_app_data_dir(app_handle: &tauri::AppHandle) -> PathBuf {
    app_handle
        .path()
        .app_data_dir()
        .expect("failed to get app data dir")
}

pub fn get_db_dir(app_handle: &tauri::AppHandle) -> PathBuf {
    get_app_data_dir(app_handle).join("db")
}

pub fn get_db_path(app_handle: &tauri::AppHandle) -> PathBuf {
    get_db_dir(app_handle).join("archive.sqlite")
}

pub fn get_log_dir(app_handle: &tauri::AppHandle) -> PathBuf {
    get_app_data_dir(app_handle).join("logs")
}

pub fn get_matrix_config_path(app_handle: &tauri::AppHandle) -> PathBuf {
    get_app_data_dir(app_handle).join("matrix-config.json")
}