import React, {useEffect, useRef, useState} from 'react';
import {X} from 'lucide-react';
import {Button} from './ui/Button';
import {api} from "../services/api.ts";
import {v4 as uuidv4} from 'uuid';

interface CreateModalProps {
    isOpen: boolean;
    onClose: () => void;
    onCreate: (data: { name: string; loader: string; version: string; ram: number; requestId?: string }) => void;
}

const CreateModal: React.FC<CreateModalProps> = ({isOpen, onClose, onCreate}) => {
    const [name, setName] = useState('');
    const [loader, setLoader] = useState('vanilla');
    const [version, setVersion] = useState('1.20.1');
    const [ram, setRam] = useState(2048);
    const [loaders, setLoaders] = useState<string[]>([]);
    const [versions, setVersions] = useState<string[]>([]);
    const [isCreating, setIsCreating] = useState(false);
    const [progressMessage, setProgressMessage] = useState('');

    const wsRef = useRef<WebSocket | null>(null);

    useEffect(() => {
        if (isOpen) {
            api.getLoaders().then(response => {
                setLoaders(response.data);
                if (response.data.length > 0) {
                    setLoader(response.data[0]);
                }
            }).catch(error => {
                console.error("Failed to fetch loaders", error);
            });
        }
    }, [isOpen]);

    useEffect(() => {
        if (loader) {
            api.getLoaderVersions(loader).then(response => {
                setVersions(response.data);
                if (response.data.length > 0) {
                    setVersion(response.data[0]);
                }
            }).catch(error => {
                console.error(`Failed to fetch versions for ${loader}`, error);
            });
        }
    }, [loader]);

    useEffect(() => {
        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
        };
    }, []);

    if (!isOpen) return null;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        const newRequestId = uuidv4();
        setIsCreating(true);
        setProgressMessage('Initializing...');

        const ws = new WebSocket(`ws://localhost:23008/ws/progress/${newRequestId}`);
        wsRef.current = ws;

        ws.onopen = () => {
            onCreate({name, loader, version, ram, requestId: newRequestId});
        };

        ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.message) {
                    setProgressMessage(data.message);
                }
                if (data.progress === 100) {
                    setIsCreating(false);
                    onClose();
                    setName('');
                    setLoader(loaders.length > 0 ? loaders[0] : 'vanilla');
                    setVersion('1.20.1');
                    setRam(2048);
                    setProgressMessage('');
                    ws.close();
                    wsRef.current = null;
                }
            } catch (e) {
                console.error("Error parsing progress message", e);
            }
        };

        ws.onerror = (e) => {
            console.error("WebSocket error", e);
            setProgressMessage('Error connecting to progress stream');
        };
    };

    return (
        <div className="modal-overlay" onClick={isCreating ? undefined : onClose}>
            <div className="modal-content" onClick={(e) => e.stopPropagation()}>
                <div className="modal-header">
                    <div className="modal-title">Create New Server</div>
                    {!isCreating && (
                        <Button variant="secondary" onClick={onClose}>
                            <X size={20}/>
                        </Button>
                    )}
                </div>
                {isCreating ? (
                    <div className="p-4 text-center">
                        <div className="mb-2">Creating server...</div>
                        <div className="text-sm text-gray-500">{progressMessage}</div>
                        <div className="mt-4 w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700">
                            <div className="bg-blue-600 h-2.5 rounded-full animate-pulse" style={{width: '100%'}}></div>
                        </div>
                    </div>
                ) : (
                    <form onSubmit={handleSubmit}>
                        <div className="form-group">
                            <label>Server Name</label>
                            <input
                                type="text"
                                className="form-input"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                placeholder="My Survival World"
                                required
                            />
                        </div>

                        <div style={{display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '15px'}}>
                            <div className="form-group">
                                <label>Loader</label>
                                <select className="form-select" value={loader}
                                        onChange={(e) => setLoader(e.target.value)}>
                                    {loaders.map(l => (
                                        <option key={l} value={l}>{l.charAt(0).toUpperCase() + l.slice(1)}</option>
                                    ))}
                                </select>
                            </div>
                            <div className="form-group">
                                <label>Version</label>
                                <select className="form-select" value={version}
                                        onChange={(e) => setVersion(e.target.value)}>
                                    {versions.map(v => (
                                        <option key={v} value={v}>{v}</option>
                                    ))}
                                </select>
                            </div>
                        </div>

                        <div className="form-group">
                            <label>RAM (MB)</label>
                            <input
                                type="number"
                                className="form-input"
                                value={ram}
                                onChange={(e) => setRam(Number(e.target.value))}
                                min="1024"
                                step="512"
                            />
                        </div>

                        <div className="modal-actions">
                            <Button type="button" variant="secondary" onClick={onClose}>Cancel</Button>
                            <Button type="submit">Create Server</Button>
                        </div>
                    </form>
                )}
            </div>
        </div>
    );
};

export default CreateModal;
