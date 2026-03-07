use std::path::Path;
use std::process::Command;

#[tauri::command]
pub fn check_file_exists(path: String) -> bool {
    Path::new(&path).exists()
}

#[tauri::command]
pub fn is_obs_process_running() -> bool {
    #[cfg(target_os = "macos")]
    {
        if let Ok(output) = Command::new("pgrep").arg("OBS").output() {
            return !output.stdout.is_empty();
        }
        false
    }
    
    #[cfg(target_os = "windows")]
    {
        if let Ok(output) = Command::new("tasklist")
            .args(["/FI", "IMAGENAME eq obs64.exe"])
            .output() 
        {
            let output_str = String::from_utf8_lossy(&output.stdout);
            return output_str.contains("obs64.exe");
        }
        false
    }
    
    #[cfg(target_os = "linux")]
    {
        if let Ok(output) = Command::new("pgrep").arg("obs").output() {
            return !output.stdout.is_empty();
        }
        false
    }
}

#[tauri::command]
pub fn quit_obs_gracefully() -> Result<(), String> {
    #[cfg(target_os = "macos")]
    {
        // On macOS, tell OBS to quit directly without needing accessibility permissions
        println!("[OBS] Sending quit command to OBS...");
        let result = Command::new("osascript")
            .args([
                "-e",
                "tell application \"OBS\" to quit"
            ])
            .output();
        
        match result {
            Ok(_) => {
                println!("[OBS] ✅ Quit command sent");
                Ok(())
            },
            Err(e) => {
                println!("[OBS] ⚠️ Failed to send quit: {}", e);
                Err(format!("Failed to quit OBS gracefully: {}", e))
            }
        }
    }
    
    #[cfg(target_os = "windows")]
    {
        // On Windows, send WM_CLOSE to OBS window
        println!("[OBS] Closing OBS window gracefully...");
        let result = Command::new("powershell")
            .args([
                "-Command",
                "(Get-Process obs64 -ErrorAction SilentlyContinue).CloseMainWindow()"
            ])
            .output();
        
        match result {
            Ok(_) => {
                println!("[OBS] ✅ Close window command sent");
                Ok(())
            },
            Err(e) => Err(format!("Failed to close OBS: {}", e))
        }
    }
    
    #[cfg(target_os = "linux")]
    {
        // On Linux, try to close window gracefully using wmctrl or xdotool
        println!("[OBS] Attempting to close OBS window...");
        
        // Try wmctrl first
        let wmctrl_result = Command::new("wmctrl")
            .args(["-c", "OBS"])
            .output();
        
        match wmctrl_result {
            Ok(_) => {
                println!("[OBS] ✅ Window close command sent");
                Ok(())
            },
            Err(e) => Err(format!("Failed to close OBS: {}", e))
        }
    }
}

#[tauri::command]
pub fn launch_application(path: String) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        // Windows: Launch OBS executable
        let path_obj = Path::new(&path);
        let working_dir = path_obj.parent()
            .ok_or_else(|| "Failed to get parent directory".to_string())?;
        
        Command::new(&path)
            .current_dir(working_dir)
            .spawn()
            .map_err(|e| format!("Failed to launch OBS: {}. Try launching OBS manually.", e))?;
        Ok(())
    }
    
    #[cfg(target_os = "macos")]
    {
        // macOS: Use 'open' command to launch .app bundle
        if path.ends_with(".app") {
            Command::new("open")
                .arg("-a")
                .arg(&path)
                .spawn()
                .map_err(|e| format!("Failed to launch OBS: {}. Try launching OBS manually.", e))?;
        } else {
            // If it's a binary path inside the app bundle
            Command::new(&path)
                .spawn()
                .map_err(|e| format!("Failed to launch OBS: {}. Try launching OBS manually.", e))?;
        }
        Ok(())
    }
    
    #[cfg(target_os = "linux")]
    {
        // Linux: Launch OBS executable directly
        Command::new(&path)
            .spawn()
            .map_err(|e| format!("Failed to launch OBS: {}. Try launching OBS manually.", e))?;
        Ok(())
    }
}
