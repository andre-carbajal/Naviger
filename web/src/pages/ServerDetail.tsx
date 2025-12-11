import React, {useCallback, useEffect, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {ArrowLeft, HardDrive, Play, Settings, Square} from 'lucide-react';
import {api} from '../services/api';
import type {Server} from '../types';
import {useConsole} from '../hooks/useConsole';
import ConsoleView from '../components/ConsoleView';
import EditServerModal from '../components/EditServerModal';

const ServerDetail: React.FC = () => {
    const {id} = useParams<{ id: string }>();
    const navigate = useNavigate();
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
        const interval = setInterval(fetchServer, 2000); // Poll for status updates
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

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'RUNNING':
                return '#4ade80';
            case 'STARTING':
                return '#facc15';
            case 'STOPPED':
                return '#f87171';
            case 'STOPPING':
                return '#f59e0b';
            default:
                return '#888';
        }
    };

    return (
        <div className="server-detail"
             style={{height: 'calc(100vh - 120px)', display: 'flex', flexDirection: 'column'}}>
            <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px'}}>
                <div style={{display: 'flex', alignItems: 'center', gap: '15px'}}>
                    <button className="btn-secondary" onClick={() => navigate('/')} style={{padding: '8px'}}>
                        <ArrowLeft size={20}/>
                    </button>
                    <div>
                        <h1 style={{margin: 0, fontSize: '1.5rem', display: 'flex', alignItems: 'center', gap: '10px'}}>
                            {server.name}
                            <span style={{
                                fontSize: '0.9rem',
                                padding: '2px 8px',
                                borderRadius: '12px',
                                backgroundColor: getStatusColor(server.status),
                                color: '#000',
                                fontWeight: 'bold'
                            }}>
                                {server.status}
                            </span>
                        </h1>
                        <div style={{color: 'var(--text-muted)', fontSize: '0.9rem'}}>
                            {server.loader} {server.version} • {server.ram}MB RAM • Port {server.port}
                        </div>
                    </div>
                </div>

                <div style={{display: 'flex', gap: '10px'}}>
                    {server.status === 'STOPPED' ? (
                        <button onClick={handleStart}
                                style={{backgroundColor: '#166534', gap: '8px', display: 'flex', alignItems: 'center'}}>
                            <Play size={18}/> Start
                        </button>
                    ) : (
                        <button onClick={handleStop}
                                style={{backgroundColor: '#991b1b', gap: '8px', display: 'flex', alignItems: 'center'}}
                                disabled={server.status === 'STARTING' || server.status === 'STOPPING'}>
                            <Square size={18}/> Stop
                        </button>
                    )}
                    <button className="btn-secondary" onClick={() => setIsEditModalOpen(true)}>
                        <Settings size={18}/>
                    </button>
                    <button className="btn-secondary">
                        <HardDrive size={18}/>
                    </button>
                </div>
            </div>

            <div style={{flex: 1, display: 'flex', flexDirection: 'column', gap: '10px', minHeight: 0}}>
                <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 5px'}}>
                    <span style={{fontWeight: 'bold'}}>Console</span>
                    <span style={{fontSize: '0.8rem', color: isConnected ? '#4ade80' : '#f87171'}}>
                        {isConnected ? '● Connected' : '○ Disconnected'}
                    </span>
                </div>

                <div style={{
                    flex: 1,
                    minHeight: 0,
                    border: '1px solid var(--border-color)',
                    borderRadius: '8px',
                    overflow: 'hidden'
                }}>
                    <ConsoleView logs={logs}/>
                </div>

                <form onSubmit={handleCommandSubmit} style={{display: 'flex', gap: '10px'}}>
                    <input
                        type="text"
                        value={commandInput}
                        onChange={(e) => setCommandInput(e.target.value)}
                        className="form-input"
                        placeholder="Type a command..."
                        style={{flex: 1}}
                        disabled={!isConnected}
                    />
                    <button type="submit" disabled={!isConnected || !commandInput.trim()}>
                        Send
                    </button>
                </form>
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
