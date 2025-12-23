import React, {useCallback, useEffect, useState} from 'react';
import {useParams} from 'react-router-dom';
import {api} from '../services/api';
import type {Backup} from '../types';
import {Button} from '../components/ui/Button';
import {Plus, RotateCcw, Trash2} from 'lucide-react';
import CreateBackupModal from '../components/CreateBackupModal';
import RestoreBackupModal from '../components/RestoreBackupModal';
import {useServers} from '../hooks/useServers';

const Backups: React.FC = () => {
    const {id} = useParams<{ id: string }>();
    const [backups, setBackups] = useState<Backup[]>([]);
    const {servers, refresh: refreshServers} = useServers();
    const [isCreateModalOpen, setCreateModalOpen] = useState(false);
    const [restoreModalOpen, setRestoreModalOpen] = useState(false);
    const [selectedBackup, setSelectedBackup] = useState<string | null>(null);

    const fetchBackups = useCallback(() => {
        const promise = id && id !== 'all' ? api.listBackups(id) : api.listAllBackups();
        promise.then(response => {
            setBackups(response.data || []);
        }).catch(error => {
            console.error("Failed to fetch backups:", error);
            setBackups([]);
        });
    }, [id]);

    useEffect(() => {
        fetchBackups();
    }, [id, fetchBackups]);

    const handleCreateBackup = async (serverId: string, name: string) => {
        await api.createBackup(serverId, name);
        fetchBackups();
    };

    const handleDelete = (backupName: string) => {
        if (window.confirm(`Are you sure you want to delete the backup "${backupName}"?`)) {
            api.deleteBackup(backupName).then(() => {
                fetchBackups();
            }).catch(error => {
                console.error("Failed to delete backup:", error);
            });
        }
    };

    const handleRestoreClick = (backupName: string) => {
        setSelectedBackup(backupName);
        setRestoreModalOpen(true);
    };

    const handleRestore = async (backupName: string, data: any) => {
        await api.restoreBackup(backupName, data);
        alert('Backup restored successfully!');
        refreshServers();
    };

    const isGlobalView = !id || id === 'all';

    return (
        <div>
            <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                <h1>Backups</h1>
                <Button onClick={() => setCreateModalOpen(true)}>
                    <Plus size={16}/> Create Backup
                </Button>
            </div>
            <div className="card">
                <table className="data-table">
                    <thead>
                    <tr>
                        <th>Name</th>
                        <th>Size</th>
                        <th>Actions</th>
                    </tr>
                    </thead>
                    <tbody>
                    {backups.map(backup => (
                        <tr key={backup.name}>
                            <td>{backup.name}</td>
                            <td>{(backup.size / 1024 / 1024).toFixed(2)} MB</td>
                            <td>
                                <div style={{display: 'flex', gap: '5px'}}>
                                    <Button variant="secondary" onClick={() => handleRestoreClick(backup.name)}>
                                        <RotateCcw size={16}/> Restore
                                    </Button>
                                    <Button variant="danger" onClick={() => handleDelete(backup.name)}>
                                        <Trash2 size={16}/> Delete
                                    </Button>
                                </div>
                            </td>
                        </tr>
                    ))}
                    {backups.length === 0 && (
                        <tr>
                            <td colSpan={3} style={{textAlign: 'center', padding: '20px', color: 'var(--text-muted)'}}>
                                No backups found.
                            </td>
                        </tr>
                    )}
                    </tbody>
                </table>
            </div>

            <CreateBackupModal
                isOpen={isCreateModalOpen}
                onClose={() => setCreateModalOpen(false)}
                onCreate={handleCreateBackup}
                servers={servers}
                defaultServerId={!isGlobalView ? id : undefined}
            />

            {selectedBackup && (
                <RestoreBackupModal
                    isOpen={restoreModalOpen}
                    onClose={() => {
                        setRestoreModalOpen(false);
                        setSelectedBackup(null);
                    }}
                    onRestore={handleRestore}
                    backupName={selectedBackup}
                    servers={servers}
                />
            )}
        </div>
    );
};

export default Backups;
