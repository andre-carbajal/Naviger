import React, {useEffect, useRef} from 'react';
import {Terminal} from 'xterm';
import {FitAddon} from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

interface ConsoleViewProps {
    logs: string[];
}

const ConsoleView: React.FC<ConsoleViewProps> = ({logs}) => {
    const terminalRef = useRef<HTMLDivElement>(null);
    const xtermRef = useRef<Terminal | null>(null);
    const fitAddonRef = useRef<FitAddon | null>(null);

    const lastLogIndexRef = useRef(0);

    useEffect(() => {
        if (!terminalRef.current) return;

        const term = new Terminal({
            theme: {
                background: '#1e1e1e',
                foreground: '#ffffff',
                cursor: '#ffffff',
            },
            fontSize: 14,
            fontFamily: 'Consolas, "Courier New", monospace',
            cursorBlink: true,
            convertEol: true,
        });

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.open(terminalRef.current);
        fitAddon.fit();

        xtermRef.current = term;
        fitAddonRef.current = fitAddon;

        const handleResize = () => {
            fitAddon.fit();
        };
        window.addEventListener('resize', handleResize);

        return () => {
            window.removeEventListener('resize', handleResize);
            term.dispose();
            xtermRef.current = null;
        };
    }, []);

    useEffect(() => {
        if (!xtermRef.current) return;

        const newLogs = logs.slice(lastLogIndexRef.current);
        if (newLogs.length > 0) {
            newLogs.forEach(line => {
                xtermRef.current?.writeln(line);
            });
            lastLogIndexRef.current = logs.length;
        }

    }, [logs]);

    useEffect(() => {
        const timeoutId = setTimeout(() => {
            fitAddonRef.current?.fit();
        }, 100);
        return () => clearTimeout(timeoutId);
    }, []);

    return (
        <div
            style={{
                height: '100%',
                width: '100%',
                backgroundColor: '#1e1e1e',
                padding: '10px',
                borderRadius: '8px',
                overflow: 'hidden'
            }}
        >
            <div ref={terminalRef} style={{height: '100%', width: '100%'}}/>
        </div>
    );
};

export default ConsoleView;
