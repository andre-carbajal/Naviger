import React, {useState} from 'react';
import {Plus, Server as ServerIcon} from 'lucide-react';
import ServerCard from '../components/ServerCard';
import CreateModal from '../components/CreateModal';
import {useServers} from '../hooks/useServers';
import {Button} from '../components/ui/Button';

const Dashboard: React.FC = () => {
    const {servers, loading, createServer, startServer, stopServer, deleteServer} = useServers();
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

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
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {servers.map(server => (
                        <ServerCard
                            key={server.id}
                            server={server}
                            onStart={startServer}
                            onStop={stopServer}
                            onDelete={deleteServer}
                        />
                    ))}
                </div>
            )}

            <CreateModal
                isOpen={isCreateModalOpen}
                onClose={() => setIsCreateModalOpen(false)}
                onCreate={createServer}
            />
        </div>
    );
};

export default Dashboard;
