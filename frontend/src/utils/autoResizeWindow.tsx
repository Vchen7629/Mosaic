import { useEffect } from "react";
import { getCurrentWindow, LogicalSize } from "@tauri-apps/api/window";

export function AutoResizeWindow() {
    useEffect(() => {
        const resize = () => {
            const height = document.body.scrollHeight;
            getCurrentWindow().setSize(new LogicalSize(360, height)).catch(console.error);
        };

        resize();

        const observer = new ResizeObserver(resize);
        observer.observe(document.body);

        return () => observer.disconnect();
    }, []);
}
