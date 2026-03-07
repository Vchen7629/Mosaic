import { invoke } from "@tauri-apps/api/core";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect } from "react";

export function handleBackendLifecycle() {
    useEffect(() => {
        invoke("start_backend_api").catch(console.error);

        const unlisten = getCurrentWindow().onCloseRequested(async (event) => {
            event.preventDefault();
            await invoke("stop_backend_api").catch(console.error);
            await getCurrentWindow().destroy();
        })

        return () => {
            unlisten.then(fn => fn())
        }
    }, [])
}