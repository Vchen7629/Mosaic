use std::net::TcpStream;
use std::process::Command;

pub fn kill_processes_on_ports() {
    let port = "8000";
    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        use std::collections::HashSet;
        const CREATE_NO_WINDOW: u32 = 0x08000000;

        println!("[Rust] Checking for processes using port {} via netstat", port);
        let cmd = format!("netstat -ano | findstr :{}", port);

        let mut netstat_cmd = Command::new("cmd");
        netstat_cmd.args(["/C", &cmd]).creation_flags(CREATE_NO_WINDOW);

        if let Ok(out) = netstat_cmd.output() {
            let text = String::from_utf8_lossy(&out.stdout).to_string();
            let mut pids_to_kill = HashSet::new();

            // Collect unique PIDs first
            for line in text.lines() {
                let parts: Vec<&str> = line.split_whitespace().collect();
                if parts.len() >= 5 {
                    let state = parts[3];
                    let pid_str = parts[4];
                    if pid_str == "0" {
                        // TIME_WAIT or similar; ignore
                        continue;
                    }
                    if let Ok(pid_val) = pid_str.parse::<u32>() {
                        if state == "LISTENING" || state == "ESTABLISHED" {
                            pids_to_kill.insert(pid_val);
                        }
                    }
                }
            }

            // Kill each unique PID once
            for pid_val in pids_to_kill {
                println!("[Rust] Found PID {} on port {} — taskkilling", pid_val, port);
                let mut kill_cmd = Command::new("taskkill");
                kill_cmd.args(["/T", "/F", "/PID", &pid_val.to_string()])
                    .creation_flags(CREATE_NO_WINDOW);
                let _ = kill_cmd.output();
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        use std::collections::HashSet;
        println!("[Rust] Checking for processes using port {} via lsof", port);
        if let Ok(out) = Command::new("sh").args(["-c", &format!("lsof -t -i:{} || true", port)]).output() {
            let text = String::from_utf8_lossy(&out.stdout).to_string();
            let mut pids_to_kill = HashSet::new();

            // Collect unique PIDs first
            for line in text.lines() {
                if let Ok(pid_val) = line.trim().parse::<u32>() {
                    pids_to_kill.insert(pid_val);
                }
            }

            // Kill each unique PID once
            for pid_val in pids_to_kill {
                println!("[Rust] Found PID {} on port {} — killing", pid_val, port);
                let _ = Command::new("kill").args(["-9", &pid_val.to_string()]).output();
            }
        }
    }
}

pub fn verify_ports_clear_and_print() {
    if !ports_in_use() {
        println!("[Rust] All services stopped successfully!");
    }
}

pub fn wait_for_port_ready(timeout: std::time::Duration) -> bool {
    let start = std::time::Instant::now();
    while start.elapsed() < timeout {
        if TcpStream::connect("127.0.0.1:8000").is_ok() {
            println!("[Rust] Port 8000 is ready");
            return true;
        }
        std::thread::sleep(std::time::Duration::from_millis(100));
    }
    println!("[Rust] Warning: port 8000 not ready after {:?}", timeout);
    false
}

pub fn wait_for_ports_free(timeout: std::time::Duration) {
    let start = std::time::Instant::now();
    while start.elapsed() < timeout {
        if !ports_in_use() {
            return;
        }
        std::thread::sleep(std::time::Duration::from_millis(200));
    }
    println!("[Rust] Warning: port 8000 still in use after wait; start may fail");
}

// Helpers for port checks and platform-specific kills. Kept small and
// private to keep the main functions concise while preserving behavior.
pub fn ports_in_use() -> bool {
    let port = "8000";
    #[cfg(target_os = "windows")]
    {
        let cmd = format!("netstat -ano | findstr :{}", port);

        let mut check_cmd = Command::new("cmd");
        check_cmd.args(["/C", &cmd]);

        #[cfg(target_os = "windows")]
        {
            use std::os::windows::process::CommandExt;
            const CREATE_NO_WINDOW: u32 = 0x08000000;
            check_cmd.creation_flags(CREATE_NO_WINDOW);
        }

        if let Ok(out) = check_cmd.output() {
            let text = String::from_utf8_lossy(&out.stdout).to_string();
            for line in text.lines() {
                // netstat format: Proto LocalAddr ForeignAddr State PID
                let parts: Vec<&str> = line.split_whitespace().collect();
                if parts.len() >= 5 {
                    let state = parts[3];
                    let pid_str = parts[4];
                    if pid_str != "0" && (state == "LISTENING" || state == "ESTABLISHED") {
                        return true;
                    }
                }
            }
        }
    }
    #[cfg(not(target_os = "windows"))]
    {
        if let Ok(o) = Command::new("sh").args(["-c", &format!("lsof -t -i:{} || true", port)]).output() {
            if !o.stdout.is_empty() {
                return true;
            }
        }
    }
    false
}
