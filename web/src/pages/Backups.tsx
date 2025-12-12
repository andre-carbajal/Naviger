import React, { useEffect, useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { api } from '../services/api';
import type { Backup, Server } from '../types';
import { Button } from '../components/ui/Button';
import { Plus, Trash2 } from 'lucide-react';
import CreateBackupModal from '../components/CreateBackupModal';

const Backups: React.FC = () => {
    const { id } = useParams<{ id: string }>();
    const [backups, setBackups] = useState<Backup[]>([]);
    const [servers, setServers] = useState<Server[]>([]);
    const [isCreateModalOpen, setCreateModalOpen] = useState(false);

    const fetchBackups = useCallback(() => {
        const promise = id && id !== 'all' ? api.listBackups(id) : api.listAllBackups();
        promise.then(response => {
            setBackups(response.data || []);
        }).catch(error => {
            console.error("Failed to fetch backups:", error);
            setBackups([]); // Ensure backups is always an array
        });
    }, [id]);

    const fetchServers = useCallback(() => {
        api.getServers().then(res => {
            setServers(res.data || []);
        }).catch(err => {
            console.error("Failed to fetch servers:", err);
        });
    }, []);

    useEffect(() => {
        fetchBackups();
        if (!id || id === 'all') {
            fetchServers();
        }
    }, [id, fetchBackups, fetchServers]);

    const handleCreateBackup = async (serverId: string, name: string) => {
        await api.createBackup(serverId, name);
        fetchBackups(); // Refresh the list after creation
    };

    const handleDelete = (backupName: string) => {
        if (window.confirm(`Are you sure you want to delete the backup "${backupName}"?`)) {
            api.deleteBackup(backupName).then(() => {
                fetchBackups(); // Refresh the list after deletion
            }).catch(error => {
                console.error("Failed to delete backup:", error);
            });
        }
    };

    const isGlobalView = !id || id === 'all';

    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h1>Backups</h1>
                <Button onClick={() => setCreateModalOpen(true)}>
                    <Plus size={16} /> Create Backup
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
                                    <Button variant="danger" onClick={() => handleDelete(backup.name)}>
                                        <Trash2 size={16} /> Delete
                                    </Button>
                                </td>
                            </tr>
                        ))}
                        {backups.length === 0 && (
                            <tr>
                                <td colSpan={3} style={{ textAlign: 'center', padding: '20px', color: 'var(--text-muted)' }}>
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
        </div>
    );
};

export default Backups;
