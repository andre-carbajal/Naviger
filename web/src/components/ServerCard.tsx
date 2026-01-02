import {Copy, Cpu, HardDrive, MemoryStick, Play, Square, Terminal, Trash2} from 'lucide-react';
import {Link} from 'react-router-dom';
import type {Server, ServerStats} from '../types';
import {Button} from './ui/Button';
import {useState} from 'react';
import {api} from '../services/api';

interface ServerCardProps {
    server: Server;
    stats?: ServerStats;
    onStart: (id: string) => void;
    onStop: (id: string) => void;
    onDelete: (id: string) => void;
}

const ServerCard: React.FC<ServerCardProps> = ({server, stats, onStart, onStop, onDelete}) => {
    const [iconError, setIconError] = useState(false);

    const formatBytes = (bytes: number) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    if (server.status === 'CREATING') {
        return (
            <div className="card">
                <div className="card-header">
                    <div>
                        <h3 className="card-title">{server.name}</h3>
                        <div className="text-sm" style={{color: 'var(--text-muted)'}}>{server.id}</div>
                    </div>
                    <span className="status-badge status-creating">CREATING</span>
                </div>
                <div className="card-content">
                    <div className="p-4">
                        <div className="mb-4 text-center font-bold">Creating server...</div>
                        <div style={{display: 'flex', flexDirection: 'column', gap: '8px'}}>
                            {server.steps && server.steps.length > 0 ? (() => {
                                const step = server.steps[server.steps.length - 1];
                                return (
                                    <div style={{display: 'flex', flexDirection: 'column', gap: '4px'}}>
                                        <div style={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            gap: '8px',
                                            fontSize: '0.9rem',
                                            justifyContent: 'flex-start'
                                        }}>
                                            {step.state === 'running' && (
                                                <span className="spinner-dot" style={{
                                                    width: '8px',
                                                    height: '8px',
                                                    borderRadius: '50%',
                                                    backgroundColor: '#3b82f6',
                                                    animation: 'pulse 1s infinite'
                                                }}></span>
                                            )}
                                            {step.state === 'failed' && <span style={{color: '#f87171'}}>✗</span>}
                                            {step.state === 'done' && <span style={{color: '#4ade80'}}>✓</span>}
                                            <span style={{
                                                color: 'var(--text-main)',
                                                fontWeight: 600
                                            }}>
                                                {step.label}
                                            </span>
                                        </div>
                                        {step.state === 'running' && step.progress !== undefined && (
                                            <div style={{
                                                marginTop: '8px',
                                                height: '4px',
                                                background: 'rgba(255,255,255,0.1)',
                                                borderRadius: '2px',
                                                overflow: 'hidden'
                                            }}>
                                                <div style={{
                                                    height: '100%',
                                                    background: '#3b82f6',
                                                    width: `${step.progress}%`,
                                                    transition: 'width 0.3s ease'
                                                }}></div>
                                            </div>
                                        )}
                                    </div>
                                );
                            })() : (
                                <div className="text-sm text-center text-gray-500">Initializing...</div>
                            )}
                        </div>
                    </div>
                </div>
                <div className="card-actions">
                    <Button disabled>
                        <Play size={16}/> Start
                    </Button>
                    <Button variant="secondary" disabled>
                        <Square size={16} fill="currentColor"/> Stop
                    </Button>
                    <Button variant="secondary" disabled>
                        <Terminal size={16}/> Console
                    </Button>
                    <Button variant="danger" disabled>
                        <Trash2 size={16}/> Delete
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className="card">
            <div className="card-header">
                <div style={{display: 'flex', alignItems: 'center', gap: '12px'}}>
                    {!iconError ? (
                        <img
                            src={api.getServerIconUrl(server.id)}
                            alt="Server Icon"
                            onError={() => setIconError(true)}
                            style={{
                                width: '48px',
                                height: '48px',
                                borderRadius: '4px',
                                objectFit: 'contain',
                                imageRendering: 'pixelated'
                            }}
                        />
                    ) : (
                        <div style={{
                            width: '48px',
                            height: '48px',
                            borderRadius: '4px',
                            backgroundColor: 'rgba(255, 255, 255, 0.1)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: '24px',
                            color: 'var(--text-muted)'
                        }}>
                            {server.name.charAt(0).toUpperCase()}
                        </div>
                    )}
                    <div>
                        <h3 className="card-title">{server.name}</h3>
                        <div style={{display: 'flex', alignItems: 'center', gap: '8px', marginTop: '4px'}}>
                        <span style={{
                            fontFamily: 'monospace',
                            background: 'rgba(0,0,0,0.3)',
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontSize: '0.8rem',
                            color: 'var(--text-muted)'
                        }}>
                            {server.id}
                        </span>
                            <button
                                onClick={() => {
                                    navigator.clipboard.writeText(server.id);
                                }}
                                className="btn-secondary"
                                style={{
                                    padding: '2px',
                                    border: 'none',
                                    cursor: 'pointer',
                                    borderRadius: '4px',
                                    display: 'flex'
                                }}
                                title="Copy ID"
                            >
                                <Copy size={12}/>
                            </button>
                        </div>
                    </div>
                </div>
                <span className={`status-badge status-${server.status.toLowerCase()}`}>{server.status}</span>
            </div>
            <div className="card-content">
                <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '15px',
                    color: 'var(--text-muted)',
                    fontSize: '0.9rem',
                    marginBottom: '15px'
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
                    <div>Port <span style={{fontFamily: 'monospace', color: 'var(--text-main)'}}>{server.port}</span>
                    </div>
                </div>

                {/* Stats Row */}
                <div style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(3, 1fr)',
                    gap: '10px',
                    padding: '10px',
                    background: 'rgba(0,0,0,0.2)',
                    borderRadius: '8px',
                    fontSize: '0.85rem'
                }}>
                    <div style={{display: 'flex', flexDirection: 'column', gap: '4px'}}>
                        <div style={{display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-muted)'}}>
                            <Cpu size={14}/> CPU
                        </div>
                        <div style={{fontWeight: 600}}>
                            {server.status === 'RUNNING' && stats ? `${stats.cpu.toFixed(1)}%` : '-'}
                        </div>
                    </div>
                    <div style={{display: 'flex', flexDirection: 'column', gap: '4px'}}>
                        <div style={{display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-muted)'}}>
                            <MemoryStick size={14}/> RAM
                        </div>
                        <div style={{fontWeight: 600}}>
                            {server.status === 'RUNNING' && stats ? `${formatBytes(stats.ram)} / ${server.ram}MB` : '-'}
                        </div>
                    </div>
                    <div style={{display: 'flex', flexDirection: 'column', gap: '4px'}}>
                        <div style={{display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-muted)'}}>
                            <HardDrive size={14}/> Disk
                        </div>
                        <div style={{fontWeight: 600}}>
                            {stats ? formatBytes(stats.disk) : '-'}
                        </div>
                    </div>
                </div>
            </div>
            <div className="card-actions">
                <Button onClick={() => onStart(server.id)} disabled={server.status === 'RUNNING'}>
                    <Play size={16}/> Start
                </Button>
                <Button variant="secondary" onClick={() => onStop(server.id)} disabled={server.status === 'STOPPED'}>
                    <Square size={16} fill="currentColor"/> Stop
                </Button>
                <Link to={`/servers/${server.id}`} className="btn btn-secondary" style={{textDecoration: 'none'}}>
                    <Terminal size={16}/> Console
                </Link>
                <Button variant="danger" onClick={() => onDelete(server.id)}>
                    <Trash2 size={16}/> Delete
                </Button>
            </div>
        </div>
    );
};

export default ServerCard;
