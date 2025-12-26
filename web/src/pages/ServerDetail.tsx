import React, {useCallback, useEffect, useState} from 'react';
import {useParams} from 'react-router-dom';
import {Play, Settings, Square} from 'lucide-react';
import {api} from '../services/api';
import type {Server} from '../types';
import {useConsole} from '../hooks/useConsole';
import ConsoleView from '../components/ConsoleView';
import EditServerModal from '../components/EditServerModal';
import {Button} from '../components/ui/Button';

const ServerDetail: React.FC = () => {
    const {id} = useParams<{ id: string }>();
    const [server, setServer] = useState<Server | null>(null);
    const [loading, setLoading] = useState(true);
    const [isEditModalOpen, setIsEditModalOpen] = useState(false);
    const [commandInput, setCommandInput] = useState('');

    const {logs, sendCommand, isConnected} = useConsole(id || '');

    const fetchServer = useCallback(async () => {
        if (!id) return;
        try {
            const res = await api.getServer(id);
            setServer(res.data);
        } catch (err) {
            console.error("Failed to fetch server:", err);
            if ((err as any).response?.status === 404) {
                setServer(null);
            }
        } finally {
            setLoading(false);
        }
    }, [id]);

    useEffect(() => {
        fetchServer();
        const interval = setInterval(fetchServer, 2000);
        return () => clearInterval(interval);
    }, [fetchServer]);

    const handleStart = async () => {
        if (!server) return;
        try {
            await api.startServer(server.id);
            setServer(prev => prev ? {...prev, status: 'STARTING'} : null);
        } catch (e) {
            console.error(e);
        }
    };

    const handleStop = async () => {
        if (!server) return;
        try {
            await api.stopServer(server.id);
            setServer(prev => prev ? {...prev, status: 'STOPPING'} : null);
        } catch (e) {
            console.error(e);
        }
    };

    const handleSaveSettings = async (data: { name: string; ram: number }) => {
        if (!server) return;
        await api.updateServer(server.id, data);
        await fetchServer();
    };

    const handleCommandSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!commandInput.trim()) return;
        sendCommand(commandInput);
        setCommandInput('');
    };

    if (loading) return <div>Loading...</div>;
    if (!server) return <div>Server not found</div>;

    return (
        <div className="server-detail">
            <div className="modal-header">
                <div>
                    <div style={{display: 'flex', alignItems: 'center', gap: '15px', marginBottom: '8px'}}>
                        <h1 style={{margin: 0}}>{server.name}</h1>
                        <span className={`status-badge status-${server.status.toLowerCase()}`}>
                            {server.status}
                        </span>
                    </div>
                    <div className="text-sm text-gray-500 mb-2">{server.id}</div>
                    <div>
                        {server.loader} {server.version} • {server.ram}MB RAM • Port {server.port}
                    </div>
                </div>

                <div style={{display: 'flex', gap: '10px'}}>
                    {server.status === 'STOPPED' ? (
                        <Button onClick={handleStart}>
                            <Play size={18}/> Start
                        </Button>
                    ) : (
                        <Button variant="danger" onClick={handleStop}
                                disabled={server.status === 'STARTING' || server.status === 'STOPPING'}>
                            <Square size={18}/> Stop
                        </Button>
                    )}
                    <Button variant="secondary" onClick={() => setIsEditModalOpen(true)}>
                        <Settings size={18}/>
                    </Button>
                </div>
            </div>

            <div style={{display: 'flex', flexDirection: 'column', gap: '15px'}}>
                <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                    <h2 style={{margin: 0, fontSize: '1.2rem'}}>Console</h2>
                    <span style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '8px',
                        fontSize: '0.9rem',
                        color: isConnected ? '#4ade80' : 'var(--text-muted)'
                    }}>
                        {isConnected ? '● Connected' : '○ Disconnected'}
                    </span>
                </div>

                <div style={{display: 'flex', flexDirection: 'column', gap: '10px'}}>
                    <ConsoleView logs={logs}/>

                    <form onSubmit={handleCommandSubmit} style={{display: 'flex', gap: '10px'}}>
                        <input
                            type="text"
                            value={commandInput}
                            onChange={(e) => setCommandInput(e.target.value)}
                            className="form-input"
                            placeholder="Type a command..."
                            disabled={!isConnected}
                            style={{flex: 1}}
                        />
                        <Button type="submit" disabled={!isConnected || !commandInput.trim()}>
                            Send
                        </Button>
                    </form>
                </div>
            </div>

            <EditServerModal
                isOpen={isEditModalOpen}
                onClose={() => setIsEditModalOpen(false)}
                onSave={handleSaveSettings}
                server={server}
            />
        </div>
    );
};

export default ServerDetail;
