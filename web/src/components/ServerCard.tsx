import React from 'react';
import {Play, Square, Terminal, Trash2} from 'lucide-react';
import {Link} from 'react-router-dom';
import type {Server} from '../types';
import {Button} from './ui/Button';

interface ServerCardProps {
    server: Server;
    onStart: (id: string) => void;
    onStop: (id: string) => void;
    onDelete: (id: string) => void;
}

const ServerCard: React.FC<ServerCardProps> = ({server, onStart, onStop, onDelete}) => {
    if (server.status === 'CREATING') {
        return (
            <div className="card">
                <div className="card-header">
                    <h3 className="card-title">{server.name}</h3>
                    <span className="status-badge status-creating">CREATING</span>
                </div>
                <div className="card-content">
                    <div className="p-4 text-center">
                        <div className="mb-2">Creating server...</div>
                        <div className="text-sm text-gray-500">{server.progressMessage || 'Initializing...'}</div>
                        <div className="mt-4 w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700"
                             style={{backgroundColor: '#374151', marginTop: '1rem'}}>
                            <div
                                className="bg-blue-600 h-2.5 rounded-full transition-all duration-500"
                                style={{width: `${server.progress || 0}%`, backgroundColor: '#3b82f6'}}
                            ></div>
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
                <h3 className="card-title">{server.name}</h3>
                <span className={`status-badge status-${server.status.toLowerCase()}`}>{server.status}</span>
            </div>
            <div className="card-content card-stats">
                <div className="stat-item">
                    <span className="stat-label">Version</span>
                    <span className="stat-value">{server.version}</span>
                </div>
                <div className="stat-item">
                    <span className="stat-label">Loader</span>
                    <span className="stat-value">{server.loader}</span>
                </div>
                <div className="stat-item">
                    <span className="stat-label">Port</span>
                    <span className="stat-value">{server.port}</span>
                </div>
                <div className="stat-item">
                    <span className="stat-label">RAM</span>
                    <span className="stat-value">{server.ram} MB</span>
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
