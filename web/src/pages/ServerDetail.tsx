import React, {useCallback, useEffect, useState} from 'react';
import {useParams} from 'react-router-dom';
import {Copy, Cpu, HardDrive, MemoryStick, Play, Settings, Square} from 'lucide-react';
import {api} from '../services/api';
import type {Server} from '../types';
import {useConsole} from '../hooks/useConsole';
import {useServerStats} from '../hooks/useServerStats';
import ConsoleView from '../components/ConsoleView';
import EditServerModal from '../components/EditServerModal';
import {Button} from '../components/ui/Button';

const ServerDetail: React.FC = () => {
    const {id} = useParams<{ id: string }>();
    const [server, setServer] = useState<Server | null>(null);
    const [loading, setLoading] = useState(true);
    const [isEditModalOpen, setIsEditModalOpen] = useState(false);
    const [commandInput, setCommandInput] = useState('');
    const [iconError, setIconError] = useState(false);
    const [iconRefreshKey, setIconRefreshKey] = useState(0);

    const {logs, sendCommand, isConnected} = useConsole(id || '');
    const {stats} = useServerStats(id || '', server?.status === 'RUNNING');

    const formatBytes = (bytes: number) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

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

    const handleSaveSettings = async (data: { name: string; ram: number; icon?: File }) => {
        if (!server) return;
        try {
            await api.updateServer(server.id, {name: data.name, ram: data.ram});

            if (data.icon) {
                await api.uploadServerIcon(server.id, data.icon);
                setIconRefreshKey(prev => prev + 1);
                setIconError(false);
            }

            await fetchServer();
        } catch (err) {
            console.error("Failed to save settings:", err);
        }
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
                <div style={{display: 'flex', alignItems: 'center', gap: '15px'}}>
                    <div style={{width: '64px', height: '64px'}}>
                        {!iconError ? (
                            <img
                                src={`${api.getServerIconUrl(server.id)}?t=${iconRefreshKey}`}
                                alt="Server Icon"
                                onError={() => setIconError(true)}
                                style={{
                                    width: '100%',
                                    height: '100%',
                                    borderRadius: '8px',
                                    objectFit: 'contain',
                                    imageRendering: 'pixelated'
                                }}
                            />
                        ) : (
                            <div style={{
                                width: '100%',
                                height: '100%',
                                borderRadius: '8px',
                                backgroundColor: 'rgba(255, 255, 255, 0.1)',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                fontSize: '32px',
                                color: 'var(--text-muted)'
                            }}>
                                {server.name.charAt(0).toUpperCase()}
                            </div>
                        )}
                    </div>
                    <div>
                        <div style={{display: 'flex', alignItems: 'center', gap: '15px', marginBottom: '8px'}}>
                            <h1 style={{margin: 0}}>{server.name}</h1>
                            <span className={`status-badge status-${server.status.toLowerCase()}`}>
                                {server.status}
                            </span>
                        </div>
                        <div className="text-sm text-gray-500 mb-2"
                             style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                            <span style={{
                                fontFamily: 'monospace',
                                background: 'rgba(0,0,0,0.3)',
                                padding: '2px 6px',
                                borderRadius: '4px'
                            }}>
                                {server.id}
                            </span>
                            <button
                                onClick={() => {
                                    navigator.clipboard.writeText(server.id);
                                }}
                                className="btn-secondary"
                                style={{
                                    padding: '4px',
                                    border: 'none',
                                    cursor: 'pointer',
                                    borderRadius: '4px',
                                    display: 'flex'
                                }}
                                title="Copy ID"
                            >
                                <Copy size={14}/>
                            </button>
                        </div>

                        <div style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: '15px',
                            color: 'var(--text-muted)',
                            fontSize: '0.9rem'
                        }}>
                            <div style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
                                <span style={{fontWeight: 600, color: 'var(--text-main)'}}>{server.loader}</span>
                                <span>{server.version}</span>
                            </div>
                            <div style={{
                                width: '4px',
                                height: '4px',
                                borderRadius: '50%',
                                backgroundColor: 'var(--text-muted)'
                            }}></div>
                            <div>Port <span
                                style={{fontFamily: 'monospace', color: 'var(--text-main)'}}>{server.port}</span></div>
                        </div>
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
                <div style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
                    gap: '15px',
                    marginBottom: '10px'
                }}>
                    <div className="card" style={{display: 'flex', alignItems: 'center', gap: '15px', padding: '15px'}}>
                        <div style={{
                            padding: '10px',
                            borderRadius: '8px',
                            background: 'rgba(59, 130, 246, 0.1)',
                            color: '#3b82f6'
                        }}>
                            <Cpu size={24}/>
                        </div>
                        <div>
                            <div style={{fontSize: '0.85rem', color: 'var(--text-muted)'}}>CPU Usage</div>
                            <div style={{fontSize: '1.2rem', fontWeight: 600}}>
                                {server.status === 'RUNNING' ? `${stats.cpu.toFixed(1)}%` : 'Offline'}
                            </div>
                        </div>
                    </div>

                    <div className="card" style={{display: 'flex', alignItems: 'center', gap: '15px', padding: '15px'}}>
                        <div style={{
                            padding: '10px',
                            borderRadius: '8px',
                            background: 'rgba(168, 85, 247, 0.1)',
                            color: '#a855f7'
                        }}>
                            <MemoryStick size={24}/>
                        </div>
                        <div>
                            <div style={{fontSize: '0.85rem', color: 'var(--text-muted)'}}>RAM Usage</div>
                            <div style={{fontSize: '1.2rem', fontWeight: 600}}>
                                {server.status === 'RUNNING' ? `${formatBytes(stats.ram)} / ${server.ram}MB` : 'Offline'}
                            </div>
                        </div>
                    </div>

                    <div className="card" style={{display: 'flex', alignItems: 'center', gap: '15px', padding: '15px'}}>
                        <div style={{
                            padding: '10px',
                            borderRadius: '8px',
                            background: 'rgba(234, 179, 8, 0.1)',
                            color: '#eab308'
                        }}>
                            <HardDrive size={24}/>
                        </div>
                        <div>
                            <div style={{fontSize: '0.85rem', color: 'var(--text-muted)'}}>Disk Usage</div>
                            <div style={{fontSize: '1.2rem', fontWeight: 600}}>
                                {formatBytes(stats.disk)}
                            </div>
                        </div>
                    </div>
                </div>

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
