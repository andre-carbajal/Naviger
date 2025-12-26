import React, {useEffect, useState} from 'react';
import {NavLink, Outlet} from 'react-router-dom';
import {ArrowUpCircle, DatabaseBackup, LayoutDashboard, Settings, Terminal} from 'lucide-react';
import '../App.css';
import {api} from "../services/api.ts";

const Layout: React.FC = () => {
    const [updateAvailable, setUpdateAvailable] = useState(false);
    const [releaseUrl, setReleaseUrl] = useState('');

    useEffect(() => {
        api.checkUpdates().then(response => {
            if (response.data.update_available) {
                setUpdateAvailable(true);
                setReleaseUrl(response.data.release_url);
            }
        }).catch(console.error);
    }, []);

    return (
        <div className="layout">
            <aside className="sidebar">
                <div className="brand">
                    <Terminal size={24}/>
                    <span>Naviger</span>
                </div>
                <nav>
                    <NavLink to="/" className={({isActive}) => isActive ? 'nav-item active' : 'nav-item'}>
                        <LayoutDashboard size={20}/>
                        <span>Dashboard</span>
                    </NavLink>
                    <NavLink to="/servers/backups/all"
                             className={({isActive}) => isActive ? 'nav-item active' : 'nav-item'}>
                        <DatabaseBackup size={20}/>
                        <span>Backups</span>
                    </NavLink>
                    <NavLink to="/settings" className={({isActive}) => isActive ? 'nav-item active' : 'nav-item'}>
                        <Settings size={20}/>
                        <span>Settings</span>
                    </NavLink>
                </nav>
                {updateAvailable && (
                    <div className="update-notification">
                        <a href={releaseUrl} target="_blank" rel="noopener noreferrer" className="update-link">
                            <ArrowUpCircle size={20}/>
                            <span>Update Available</span>
                        </a>
                    </div>
                )}
            </aside>
            <main className="content">
                <header className="topbar">
                    <div className="user-info">Admin</div>
                </header>
                <div className="page-content">
                    <Outlet/>
                </div>
            </main>
        </div>
    );
};

export default Layout;
