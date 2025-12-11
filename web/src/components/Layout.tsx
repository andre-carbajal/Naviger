import React from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { LayoutDashboard, Server, Settings, Terminal } from 'lucide-react';
import '../App.css';

const Layout: React.FC = () => {
    return (
        <div className="layout">
            <aside className="sidebar">
                <div className="brand">
                    <Terminal size={24} />
                    <span>MC Manager</span>
                </div>
                <nav>
                    <NavLink to="/" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
                        <LayoutDashboard size={20} />
                        <span>Dashboard</span>
                    </NavLink>
                    <NavLink to="/servers" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
                        <Server size={20} />
                        <span>Servers</span>
                    </NavLink>
                    <NavLink to="/settings" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
                        <Settings size={20} />
                        <span>Settings</span>
                    </NavLink>
                </nav>
            </aside>
            <main className="content">
                <header className="topbar">
                    <div className="user-info">Admin</div>
                </header>
                <div className="page-content">
                    <Outlet />
                </div>
            </main>
        </div>
    );
};

export default Layout;
