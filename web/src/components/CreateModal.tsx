import React, {useState} from 'react';
import {X} from 'lucide-react';

interface CreateModalProps {
    isOpen: boolean;
    onClose: () => void;
    onCreate: (data: { name: string; loader: string; version: string; ram: number }) => void;
}

const CreateModal: React.FC<CreateModalProps> = ({isOpen, onClose, onCreate}) => {
    const [name, setName] = useState('');
    const [loader, setLoader] = useState('vanilla');
    const [version, setVersion] = useState('1.20.1');
    const [ram, setRam] = useState(2048);

    if (!isOpen) return null;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onCreate({name, loader, version, ram});
        onClose();
        setName('');
        setLoader('vanilla');
        setVersion('1.20.1');
        setRam(2048);
    };

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={(e) => e.stopPropagation()}>
                <div className="modal-header">
                    <div className="modal-title">Create New Server</div>
                    <button className="btn-secondary" onClick={onClose} style={{border: 'none', padding: '5px'}}>
                        <X size={20}/>
                    </button>
                </div>
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
                            <select className="form-select" value={loader} onChange={(e) => setLoader(e.target.value)}>
                                <option value="vanilla">Vanilla</option>
                                <option value="paper">Paper</option>
                                <option value="fabric">Fabric</option>
                                <option value="forge">Forge</option>
                            </select>
                        </div>
                        <div className="form-group">
                            <label>Version</label>
                            <input
                                type="text"
                                className="form-input"
                                value={version}
                                onChange={(e) => setVersion(e.target.value)}
                                placeholder="1.21"
                                required
                            />
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
                        <button type="button" className="btn-secondary" onClick={onClose}>Cancel</button>
                        <button type="submit">Create Server</button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default CreateModal;
