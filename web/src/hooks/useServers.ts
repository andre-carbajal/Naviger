import {useCallback, useEffect, useState} from 'react';
import {api} from '../services/api';
import type {Server} from '../types';

export const useServers = () => {
    const [servers, setServers] = useState<Server[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchServers = useCallback(async () => {
        try {
            const response = await api.getServers();
            setServers(response.data || []);
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    }, []);

    const createServer = async (data: { name: string; loader: string; version: string; ram: number }) => {
        try {
            await api.createServer(data);
            await fetchServers();
            return true;
        } catch (err) {
            console.error(err);
            throw err;
        }
    };

    const startServer = async (id: string) => {
        try {
            setServers(prev => prev.map(s => s.id === id ? {...s, status: 'STARTING'} : s));
            await api.startServer(id);
        } catch (err) {
            console.error(err);
            await fetchServers();
        }
    };

    const stopServer = async (id: string) => {
        try {
            await api.stopServer(id);
            await fetchServers();
        } catch (err) {
            console.error(err);
        }
    };

    const deleteServer = async (id: string) => {
        try {
            setServers(prev => prev.filter(s => s.id !== id));
            await api.deleteServer(id);
        } catch (err) {
            console.error(err);
            await fetchServers();
        }
    };

    useEffect(() => {
        fetchServers();
        const interval = setInterval(fetchServers, 5000);
        return () => clearInterval(interval);
    }, [fetchServers]);

    return {servers, loading, createServer, startServer, stopServer, deleteServer, refresh: fetchServers};
};
