use std::process::Command;
use std::path::PathBuf;
use tauri::Emitter;
use std::fs;
use crate::path_utils;

fn get_setup_flag_path(app_handle: &tauri::AppHandle) -> PathBuf {
    path_utils::get_app_data_dir(app_handle).join(".setup_complete")
}

pub fn is_setup_complete(app_handle: &tauri::AppHandle) -> bool {
    get_setup_flag_path(app_handle).exists()
}

#[tauri::command]
pub fn check_setup_status(app_handle: tauri::AppHandle) -> Result<bool, String> {
    Ok(is_setup_complete(&app_handle))
}

#[tauri::command]
pub async fn run_first_time_setup(app_handle: tauri::AppHandle) -> Result<String, String> {
    println!("[Rust] Starting first-time setup...");

    // Create a channel to send progress updates from the blocking thread
    let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel::<String>();

    // Clone app_handle for the background thread
    let app_handle_clone = app_handle.clone();

    // Spawn a task to listen for progress updates and emit events
    let app_handle_emitter = app_handle.clone();
    tokio::spawn(async move {
        while let Some(progress) = rx.recv().await {
            println!("[Rust] Emitting progress via channel: {}", progress);
            if let Err(e) = app_handle_emitter.emit("setup-progress", &progress) {
                println!("[Rust] Failed to emit event: {}", e);
            }
        }
    });

    // Run the blocking setup work in a background thread
    let result = tokio::task::spawn_blocking(move || {
        let backend_path = path_utils::get_backend_path(&app_handle)?;

        println!("[Rust] Backend path: {}", backend_path.display());
        
        // Step 1: Check if user has uv and install uv if not found
        println!("[Rust] Sending progress: checking and installing uv");
        let _ = tx.send("setting up UV".to_string());

        let uv_check = Command::new("uv")
            .arg("--version")
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();

        let use_uv = uv_check.is_ok();

        if !use_uv {
            println!("[Rust] uv not found, installing uv...");
            let _ = tx.send("uv not found, installing uv...".to_string());
            #[cfg(target_os = "windows")]
            {
                let mut install_cmd = Command::new("powershell");
                install_cmd.args(&[
                    "-ExecutionPolicy",
                    "ByPass",
                    "-Command",
                    "irm https://astral.sh/uv/install.ps1 | iex"
                ]);

                #[cfg(target_os = "windows")]
                {
                    use std::os::windows::process::CommandExt;
                    const CREATE_NO_WINDOW: u32 = 0x08000000;
                    install_cmd.creation_flags(CREATE_NO_WINDOW);
                }

                let result = install_cmd
                    .stdout(Stdio::piped())
                    .stderr(Stdio::piped())
                    .status();

                match result {
                    Ok(status) if status.success() => {
                        println!("[Rust] uv installed successfully");
                    }
                    _ => {
                        return Err("Failed to install uv".to_string());
                    }
                }
            }

            #[cfg(not(target_os = "windows"))]
            {
                let mut install_cmd = Command::new("sh");
                install_cmd.args(&[
                    "-c",
                    "curl -LsSf https://astral.sh/uv/install.sh | sh"
                ]);

                #[cfg(target_os = "windows")]
                {
                    use std::os::windows::process::CommandExt;
                    const CREATE_NO_WINDOW: u32 = 0x08000000;
                    install_cmd.creation_flags(CREATE_NO_WINDOW);
                }

                let result = install_cmd
                    .stdout(Stdio::piped())
                    .stderr(Stdio::piped())
                    .status();

                match result {
                    Ok(status) if status.success() => {
                        println!("[Rust] uv installed successfully");
                    }
                    _ => {
                        return Err("Failed to install uv".to_string());
                    }
                }
            }
        }
        
        // Step 2: Run uv sync to create venv and install dependencies
        // Use GPU version (CUDA) for Windows/Linux, CPU version for macOS
        // Send progress update via channel
        println!("[Rust] Sending progress: downloading_dependencies");
        let _ = tx.send("UV successfully setup, installing python packages...".to_string());

        let mut uv_cmd = Command::new("uv");

        println!("[Rust] Running uv sync...");
        uv_cmd
            .args(&["sync", "--locked"])
            .current_dir(&backend_path)
            .stdout(Stdio::null())
            .stderr(Stdio::null());

        #[cfg(target_os = "windows")]
        {
            use std::os::windows::process::CommandExt;
            const CREATE_NO_WINDOW: u32 = 0x08000000;
            uv_cmd.creation_flags(CREATE_NO_WINDOW);
        }

        let uv_result = uv_cmd.status();

        match uv_result {
            Ok(status) if status.success() => {
                println!("[Rust] uv sync completed successfully");
            }
            Ok(status) => {
                return Err(format!("uv sync failed with exit code: {}", status));
            }
            Err(e) => {
                return Err(format!("Failed to run uv sync: {}. Make sure 'uv' is installed and in PATH.", e));
            }
        }

        // Step 3: Download models
        println!("[Rust] Python packages installed, Downloading models...");
        let _ = tx.send("Python packages installed, Downloading models...".to_string());

        #[cfg(target_os = "windows")]
        let python_exe = backend_path.join(".venv").join("Scripts").join("python.exe");
        #[cfg(not(target_os = "windows"))]
        let python_exe = backend_path.join(".venv").join("bin").join("python3");

        use std::io::{BufRead, BufReader};
        use std::process::Stdio;

        let mut download_cmd = Command::new(&python_exe);
        download_cmd
            .arg("src/translation/download_models.py")
            .current_dir(&backend_path)
            .env("PRODUCTION", "true")
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());

        #[cfg(target_os = "windows")]
        {
            use std::os::windows::process::CommandExt;
            const CREATE_NO_WINDOW: u32 = 0x08000000;
            download_cmd.creation_flags(CREATE_NO_WINDOW);
        }

        let mut child = download_cmd
            .spawn()
            .map_err(|e| format!("Failed to spawn model download process: {}", e))?;

        // Track which model we're on
        let mut current_model = 1;

        // Spawn thread to read stderr (to avoid deadlock)
        let stderr_handle = if let Some(stderr) = child.stderr.take() {
            Some(std::thread::spawn(move || {
                let reader = BufReader::new(stderr);
                let mut errors = String::new();
                for line in reader.lines() {
                    if let Ok(line_text) = line {
                        println!("[Rust] STDERR: {}", line_text);
                        errors.push_str(&line_text);
                        errors.push('\n');
                    }
                }
                errors
            }))
        } else {
            None
        };

        // Read stdout line by line and emit progress
        if let Some(stdout) = child.stdout.take() {
            let reader = BufReader::new(stdout);
            for line in reader.lines() {
                if let Ok(line_text) = line {
                    println!("[Rust] {}", line_text);

                    // Detect model download progress based on Python script output
                    if line_text.contains("[1/3]") && current_model == 1 {
                        println!("[Rust] Sending progress: downloading_model_1");
                        let _ = tx.send("downloading model_1...".to_string());
                        current_model = 2;
                    } else if line_text.contains("[2/3]") && current_model == 2 {
                        println!("[Rust] Sending progress: downloading_model_2");
                        let _ = tx.send("downloading model_2...".to_string());
                        current_model = 3;
                    } else if line_text.contains("[3/3]") && current_model == 3 {
                        println!("[Rust] Sending progress: downloading_model_3");
                        let _ = tx.send("downloading model_3...".to_string());
                        current_model = 4;
                    }
                }
            }
        }

        let status = child.wait().map_err(|e| format!("Failed to wait for model download: {}", e))?;

        // Get stderr output
        let error_output = if let Some(handle) = stderr_handle {
            handle.join().unwrap_or_default()
        } else {
            String::new()
        };

        if !status.success() {
            let error_msg = if !error_output.is_empty() {
                format!("Model download failed:\n{}", error_output)
            } else {
                format!("Model download failed with exit code: {}", status)
            };
            return Err(error_msg);
        }

        println!("[Rust] Models downloaded successfully");

        // Step 4: Initialize database
        println!("[Rust] Initializing database...");
        let _ = tx.send("Models downloaded successfully, initializing database...".to_string());

        // Create database directory in app data
        let db_dir = path_utils::get_db_dir(&app_handle_clone);
        let db_path = path_utils::get_db_path(&app_handle_clone);

        // Create the db directory if it doesn't exist
        if !db_dir.exists() {
            fs::create_dir_all(&db_dir)
                .map_err(|e| format!("Failed to create database directory: {}", e))?;
        }

        println!("[Rust] Database will be created at: {}", db_path.display());

        // Run alembic upgrade head to create/migrate database
        let workspace_root = backend_path
            .parent()
            .expect("failed to get workspace root from backend path");

        #[cfg(target_os = "windows")]
        let python_exe = backend_path.join(".venv").join("Scripts").join("python.exe");
        #[cfg(not(target_os = "windows"))]
        let python_exe = backend_path.join(".venv").join("bin").join("python3");

        let mut alembic_cmd = Command::new(&python_exe);
        alembic_cmd
            .args(&["-m", "alembic", "-c", "backend/src/db/alembic.ini", "upgrade", "head"])
            .current_dir(workspace_root)  // Run from project root
            .env("PRODUCTION", "true")
            .env("DB_PATH", db_path.to_str().unwrap())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());

        #[cfg(target_os = "windows")]
        {
            use std::os::windows::process::CommandExt;
            const CREATE_NO_WINDOW: u32 = 0x08000000;
            alembic_cmd.creation_flags(CREATE_NO_WINDOW);
        }

        let output = alembic_cmd.output()
            .map_err(|e| format!("Failed to run database initialization: {}", e))?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            let stdout = String::from_utf8_lossy(&output.stdout);
            println!("[Rust] Alembic STDOUT: {}", stdout);
            println!("[Rust] Alembic STDERR: {}", stderr);
            return Err(format!("Database initialization failed with exit code: {}\nSTDERR: {}\nSTDOUT: {}",
                output.status, stderr, stdout));
        }

        println!("[Rust] Database initialized successfully");

        println!("[Rust] Database initialization completed");

        // Send finalizing progress
        let _ = tx.send("Database initialized successfully, finalizing...".to_string());

        println!("[Rust] First-time setup completed successfully!");
        println!("[Rust] Setup flag will be written after Matrix configuration");

        // Send complete progress event
        let _ = tx.send("complete".to_string());

        Ok("Setup completed successfully".to_string())
    })
    .await
    .map_err(|e| format!("Setup thread panicked: {}", e))??;

    Ok(result)
}

#[tauri::command]
pub fn mark_setup_complete(app_handle: tauri::AppHandle) -> Result<String, String> {
    println!("[Rust] Marking setup as complete");

    // Write setup complete flag
    let flag_path = get_setup_flag_path(&app_handle);
    if let Some(parent) = flag_path.parent() {
        let _ = fs::create_dir_all(parent);
    }
    fs::write(&flag_path, "setup_complete")
        .map_err(|e| format!("Failed to write setup flag: {}", e))?;

    println!("[Rust] Setup marked as complete");

    Ok("Setup marked as complete".to_string())
}