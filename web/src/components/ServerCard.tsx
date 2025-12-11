import React from 'react';
import {useNavigate} from 'react-router-dom';
import {Play, Square, Terminal as TerminalIcon, Trash2} from 'lucide-react';
import type {Server} from '../types';
import '../App.css';

interface ServerCardProps {
    server: Server;
    onStart: (id: string) => void;
    onStop: (id: string) => void;
    onDelete: (id: string) => void;
}

const ServerCard: React.FC<ServerCardProps> = ({server, onStart, onStop, onDelete}) => {
    const navigate = useNavigate();

    const handleCardClick = () => {
        navigate(`/servers/${server.id}`);
    };

    const handleAction = (e: React.MouseEvent, action: () => void) => {
        e.stopPropagation();
        action();
    };

    const handleDelete = (e: React.MouseEvent) => {
        e.stopPropagation();
        if (window.confirm(`Are you sure you want to delete the server "${server.name}"? This action cannot be undone.`)) {
            onDelete(server.id);
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'RUNNING':
                return '#4ade80';
            case 'STARTING':
                return '#facc15';
            case 'STOPPED':
                return '#f87171';
            default:
                return '#888';
        }
    };

    return (
        <div className="card server-card" onClick={handleCardClick}
             style={{cursor: 'pointer', display: 'flex', flexDirection: 'column', gap: '15px'}}>
            <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                <h3 style={{margin: 0, fontSize: '1.2rem', display: 'flex', alignItems: 'center', gap: '8px'}}>
                    <div style={{
                        width: '10px',
                        height: '10px',
                        borderRadius: '50%',
                        backgroundColor: getStatusColor(server.status)
                    }}></div>
                    {server.name}
                </h3>
                <span style={{
                    fontSize: '0.8rem',
                    color: 'var(--text-muted)',
                    border: '1px solid var(--border-color)',
                    padding: '2px 8px',
                    borderRadius: '4px'
                }}>
                    {server.loader} {server.version}
                </span>
            </div>

            <div style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: '10px',
                fontSize: '0.9rem',
                color: 'var(--text-muted)'
            }}>
                <div>Port: <span style={{color: 'white'}}>{server.port}</span></div>
                <div>RAM: <span style={{color: 'white'}}>{server.ram} MB</span></div>
            </div>

            <div style={{display: 'flex', gap: '10px', marginTop: 'auto'}}>
                {server.status === 'STOPPED' ? (
                    <button
                        onClick={(e) => handleAction(e, () => onStart(server.id))}
                        style={{
                            flex: 1,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            gap: '6px',
                            backgroundColor: '#166534'
                        }}
                    >
                        <Play size={16}/> Start
                    </button>
                ) : (
                    <button
                        onClick={(e) => handleAction(e, () => onStop(server.id))}
                        disabled={server.status === 'STARTING'}
                        style={{
                            flex: 1,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            gap: '6px',
                            backgroundColor: server.status === 'STARTING' ? '#854d0e' : '#991b1b',
                            cursor: server.status === 'STARTING' ? 'not-allowed' : 'pointer',
                            opacity: server.status === 'STARTING' ? 0.7 : 1
                        }}
                    >
                        <Square size={16}/> Stop
                    </button>
                )}
                <button
                    className="btn-secondary"
                    onClick={(e) => handleAction(e, () => navigate(`/servers/${server.id}`))}
                    style={{padding: '0.6em'}}
                    title="Open Console"
                >
                    <TerminalIcon size={18}/>
                </button>
                <button
                    className="btn-danger"
                    onClick={handleDelete}
                    style={{padding: '0.6em'}}
                    title="Delete Server"
                >
                    <Trash2 size={18}/>
                </button>
            </div>
        </div>
    );
};

export default ServerCard;
