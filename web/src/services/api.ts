import axios from 'axios';
import type {Backup, Server} from '../types';

const apiInstance = axios.create({
    baseURL: 'http://localhost:8080',
    timeout: 5000,
    headers: {
        'Content-Type': 'application/json',
    }
});

apiInstance.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.code === "ERR_NETWORK") {
            const event = new CustomEvent('network-error', {
                detail: {message: 'Failed to connect to the server. Please check your connection and try again.'}
            });
            window.dispatchEvent(event);
        }
        return Promise.reject(error);
    }
);

export const api = {
    getServers: () => apiInstance.get<Server[]>('/servers'),
    getServer: (id: string) => apiInstance.get<Server>(`/servers/${id}`),
    createServer: (data: Omit<Server, 'id' | 'status' | 'port' | 'pid'>) => apiInstance.post<Server>('/servers', data),
    updateServer: (id: string, data: {
        name?: string,
        ram?: number
    }) => apiInstance.put<Server>(`/servers/${id}`, data),
    deleteServer: (id: string) => apiInstance.delete(`/servers/${id}`),
    startServer: (id: string) => apiInstance.post(`/servers/${id}/start`),
    stopServer: (id: string) => apiInstance.post(`/servers/${id}/stop`),
    getPortRange: () => apiInstance.get('/settings/port-range'),
    updatePortRange: (data: { start: number, end: number }) => apiInstance.put('/settings/port-range', data),
    listBackups: (serverId: string) => apiInstance.get<Backup[]>(`/servers/${serverId}/backups`),
    listAllBackups: () => apiInstance.get<Backup[]>('/backups'),
    createBackup: (serverId: string, name?: string) => apiInstance.post(`/servers/${serverId}/backup`, {name}),
    deleteBackup: (backupName: string) => apiInstance.delete(`/backups/${backupName}`),
};
