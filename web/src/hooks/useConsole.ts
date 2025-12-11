import {useEffect, useRef, useState} from 'react';

export const useConsole = (serverId: string) => {
    const ws = useRef<WebSocket | null>(null);
    const [logs, setLogs] = useState<string[]>([]);
    const [isConnected, setIsConnected] = useState(false);

    useEffect(() => {
        if (!serverId) return;

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = 'localhost:8080';
        const url = `${protocol}//${host}/ws/servers/${serverId}/console`;

        console.log(`Connecting to WS: ${url}`);
        ws.current = new WebSocket(url);

        ws.current.onopen = () => {
            console.log('WS Connected');
            setIsConnected(true);
        };

        ws.current.onmessage = (event) => {
            setLogs((prev) => [...prev, event.data]);
        };

        ws.current.onclose = () => {
            console.log('WS Closed');
            setIsConnected(false);
        };

        ws.current.onerror = (error) => {
            console.error('WS Error:', error);
            setIsConnected(false);
        };

        return () => {
            ws.current?.close(1000, 'Component unmounted');
        };
    }, [serverId]);

    const sendCommand = (cmd: string) => {
        if (ws.current && ws.current.readyState === WebSocket.OPEN) {
            ws.current.send(cmd + '\n');
        } else {
            console.warn('WebSocket not connected, cannot send command');
        }
    };

    const clearLogs = () => {
        setLogs([]);
    }

    return {logs, sendCommand, isConnected, clearLogs};
};
