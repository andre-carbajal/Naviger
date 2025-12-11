import React, {useState} from 'react';
import {Plus, Server as ServerIcon} from 'lucide-react';
import ServerCard from '../components/ServerCard';
import CreateModal from '../components/CreateModal';
import {useServers} from '../hooks/useServers';

const Dashboard: React.FC = () => {
    const {servers, loading, createServer, startServer, stopServer, deleteServer} = useServers();
    const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

    if (loading && servers.length === 0) {
        return <div style={{display: 'flex', justifyContent: 'center', marginTop: '50px'}}>Loading servers...</div>;
    }

    return (
        <div className="dashboard">
            <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '30px'}}>
                <h1 style={{margin: 0}}>My Servers</h1>
                <button onClick={() => setIsCreateModalOpen(true)}
                        style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                    <Plus size={20}/> Create Server
                </button>
            </div>

            {servers.length === 0 && !loading ? (
                <div style={{
                    textAlign: 'center',
                    padding: '50px',
                    backgroundColor: 'var(--bg-card)',
                    borderRadius: '12px',
                    border: '1px solid var(--border-color)',
                    color: 'var(--text-muted)'
                }}>
                    <ServerIcon size={48} style={{marginBottom: '15px', opacity: 0.5}}/>
                    <p>No servers found. Create your first server to get started!</p>
                </div>
            ) : (
                <div style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
                    gap: '20px'
                }}>
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
