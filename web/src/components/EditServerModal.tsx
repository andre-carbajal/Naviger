import React, {useEffect, useState} from 'react';
import type {Server} from '../types';

interface EditServerModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (data: { name: string; ram: number }) => Promise<void>;
    server: Server | null;
}

const EditServerModal: React.FC<EditServerModalProps> = ({isOpen, onClose, onSave, server}) => {
    const [name, setName] = useState('');
    const [ram, setRam] = useState(0);
    const [isSaving, setIsSaving] = useState(false);

    useEffect(() => {
        if (isOpen && server) {
            setName(server.name);
            setRam(server.ram);
        }
    }, [isOpen]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setIsSaving(true);
        try {
            await onSave({name, ram});
            onClose();
        } catch (error) {
            console.error("Failed to save server settings:", error);
        } finally {
            setIsSaving(false);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="modal-overlay">
            <div className="modal-content" style={{maxWidth: 400}}>
                <h2>Edit Server</h2>
                <form onSubmit={handleSubmit}>
                    <div className="form-group">
                        <label>Server Name</label>
                        <input
                            type="text"
                            className="form-input"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            required
                        />
                    </div>
                    <div className="form-group">
                        <label>RAM (MB)</label>
                        <input
                            type="number"
                            className="form-input"
                            value={ram}
                            onChange={(e) => setRam(parseInt(e.target.value, 10))}
                            required
                            min="1024"
                            step="512"
                        />
                    </div>
                    <div className="modal-actions">
                        <button type="button" className="btn-secondary" onClick={onClose} disabled={isSaving}>
                            Cancel
                        </button>
                        <button type="submit" className="btn-primary" disabled={isSaving}>
                            {isSaving ? 'Saving...' : 'Save'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default EditServerModal;
