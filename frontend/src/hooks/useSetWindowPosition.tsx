import { currentMonitor, getCurrentWindow, LogicalPosition } from "@tauri-apps/api/window";
import { useEffect } from "react";

/**
 * Custom hook that puts the application in the top right of the user's window
 * since this is an overlay
 */
export function useSetWindowPosition() {
    useEffect(() => {
        const positionTopRight = async () => {
            const win = getCurrentWindow();
            const monitor = await currentMonitor();
            if (!monitor) return;
            const { width: physicalWidth } = monitor.size;
            const scale = monitor.scaleFactor;
            const logicalScreenWidth = physicalWidth / scale;
            const windowWidth = 360;
            await win.setPosition(new LogicalPosition(logicalScreenWidth - windowWidth, 0));
        };
        positionTopRight().catch(console.error);
    }, []);
}