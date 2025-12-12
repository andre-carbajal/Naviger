import React, {useEffect, useState} from 'react';
import type {Server} from '../types';
import {Button} from './ui/Button';

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
            <div className="modal-content">
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
                        <Button type="button" variant="secondary" onClick={onClose} disabled={isSaving}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={isSaving}>
                            {isSaving ? 'Saving...' : 'Save'}
                        </Button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default EditServerModal;
