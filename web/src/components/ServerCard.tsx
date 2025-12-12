import React from 'react';
import { Play, Square, Trash2, Terminal } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { Server } from '../types';
import { Button } from './ui/Button';

interface ServerCardProps {
    server: Server;
    onStart: (id: string) => void;
    onStop: (id: string) => void;
    onDelete: (id: string) => void;
}

const ServerCard: React.FC<ServerCardProps> = ({ server, onStart, onStop, onDelete }) => {
    return (
        <div className="card">
            <div className="card-header">
                <h3 className="card-title">{server.name}</h3>
                <span className={`status-badge status-${server.status.toLowerCase()}`}>{server.status}</span>
            </div>
            <div className="card-content">
                <p>Version: {server.version}</p>
                <p>RAM: {server.ram} MB</p>
            </div>
            <div className="card-actions">
                <Button onClick={() => onStart(server.id)} disabled={server.status === 'RUNNING'}>
                    <Play size={16} /> Start
                </Button>
                <Button variant="secondary" onClick={() => onStop(server.id)} disabled={server.status === 'STOPPED'}>
                    <Square size={16} fill="currentColor" /> Stop
                </Button>
                <Link to={`/servers/${server.id}`} className="btn btn-secondary" style={{ textDecoration: 'none' }}>
                    <Terminal size={16} /> Console
                </Link>
                <Button variant="danger" onClick={() => onDelete(server.id)}>
                    <Trash2 size={16} /> Delete
                </Button>
            </div>
        </div>
    );
};

export default ServerCard;
