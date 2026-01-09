import axios from 'axios';
import type {Backup, Server, ServerStats} from '../types';

const API_PORT = import.meta.env.VITE_API_PORT || 23008;
const API_HOST = window.location.hostname;
const API_PROTOCOL = window.location.protocol;
export const WS_HOST = `${API_HOST}:${API_PORT}`;

const apiInstance = axios.create({
    baseURL: `${API_PROTOCOL}//${API_HOST}:${API_PORT}`,
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
    getLoaders: () => apiInstance.get<string[]>('/loaders'),
    getLoaderVersions: (loader: string) => apiInstance.get<string[]>(`/loaders/${loader}/versions`),
    getServers: () => apiInstance.get<Server[]>('/servers'),
    getServer: (id: string) => apiInstance.get<Server>(`/servers/${id}`),
    getServerStats: (id: string) => apiInstance.get<ServerStats>(`/servers/${id}/stats`),
    getAllServerStats: () => apiInstance.get<Record<string, ServerStats>>('/servers-stats'),
    getServerIconUrl: (id: string) => `${API_PROTOCOL}//${API_HOST}:${API_PORT}/servers/${id}/icon`,
    uploadServerIcon: (id: string, file: File) => {
        const formData = new FormData();
        formData.append('icon', file);
        return apiInstance.post(`/servers/${id}/icon`, formData, {
            headers: {
                'Content-Type': 'multipart/form-data'
            }
        });
    },
    createServer: (data: Omit<Server, 'id' | 'status' | 'port'> & {
        requestId?: string
    }) => apiInstance.post<Server>('/servers', data),
    updateServer: (id: string, data: {
        name?: string,
        ram?: number,
        customArgs?: string
    }) => apiInstance.put<Server>(`/servers/${id}`, data),
    deleteServer: (id: string) => apiInstance.delete(`/servers/${id}`),
    startServer: (id: string) => apiInstance.post(`/servers/${id}/start`),
    stopServer: (id: string) => apiInstance.post(`/servers/${id}/stop`),
    getPortRange: () => apiInstance.get('/settings/port-range'),
    updatePortRange: (data: { start: number, end: number }) => apiInstance.put('/settings/port-range', data),
    listBackups: (serverId: string) => apiInstance.get<Backup[]>(`/servers/${serverId}/backups`),
    listAllBackups: () => apiInstance.get<Backup[]>('/backups'),
    createBackup: (serverId: string, name?: string, requestId?: string) => apiInstance.post<{
        status: string,
        id: string
    }>(`/servers/${serverId}/backup`, {name, requestId}),
    deleteBackup: (backupName: string) => apiInstance.delete(`/backups/${backupName}`),
    cancelBackupCreation: (requestId: string) => apiInstance.delete(`/backups/progress/${requestId}`),
    restoreBackup: (backupName: string, data: {
        targetServerId?: string,
        newServerName?: string,
        newServerRam?: number,
        newServerLoader?: string,
        newServerVersion?: string
    }) => apiInstance.post(`/backups/${backupName}/restore`, data),
    checkUpdates: () => apiInstance.get<{
        current_version: string;
        latest_version: string;
        update_available: boolean;
        release_url: string;
    }>('/updates'),
    restartDaemon: () => apiInstance.post('/system/restart'),
};
