import React, {useState} from 'react';
import {Plus, Server as ServerIcon} from 'lucide-react';
import ServerCard from '../components/ServerCard';
import CreateModal from '../components/CreateModal';
import ConfirmationModal from '../components/ConfirmationModal';
import {useServers} from '../hooks/useServers';
import {Button} from '../components/ui/Button';

const Dashboard: React.FC = () => {
    const {servers, loading, createServer, startServer, stopServer, deleteServer} = useServers();
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
    const [serverToDelete, setServerToDelete] = useState<string | null>(null);

    const handleDelete = (id: string) => {
        setServerToDelete(id);
    };

    const confirmDelete = async () => {
        if (serverToDelete) {
            await deleteServer(serverToDelete);
            setServerToDelete(null);
        }
    };

    if (loading && servers.length === 0) {
        return <div>Loading servers...</div>;
    }

    return (
        <div className="dashboard">
            <div className="modal-header">
                <h1>My Servers</h1>
                <Button onClick={() => setIsCreateModalOpen(true)}>
                    <Plus size={20}/> Create Server
                </Button>
            </div>

            {servers.length === 0 && !loading ? (
                <div className="card">
                    <ServerIcon size={48}/>
                    <p>No servers found. Create your first server to get started!</p>
                </div>
            ) : (
                <div className="servers-grid">
                    {servers.map(server => (
                        <ServerCard
                            key={server.id}
                            server={server}
                            onStart={startServer}
                            onStop={stopServer}
                            onDelete={handleDelete}
                        />
                    ))}
                </div>
            )}

            <CreateModal
                isOpen={isCreateModalOpen}
                onClose={() => setIsCreateModalOpen(false)}
                onCreate={createServer}
            />

            <ConfirmationModal
                isOpen={!!serverToDelete}
                onClose={() => setServerToDelete(null)}
                onConfirm={confirmDelete}
                title="Delete Server"
                message="Are you sure you want to delete this server? This action cannot be undone and all server files will be permanently lost."
                confirmText="Delete Server"
                isDangerous={true}
            />
        </div>
    );
};

export default Dashboard;
