import axios from 'axios';
import type { Server } from '../types';

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
                detail: { message: 'Failed to connect to the server. Please check your connection and try again.' }
            });
            window.dispatchEvent(event);
        }
        return Promise.reject(error);
    }
);

export const api = {
    getServers: () => apiInstance.get<Server[]>('/servers'),
    createServer: (data: Omit<Server, 'id' | 'status' | 'port' | 'pid'>) => apiInstance.post<Server>('/servers', data),
    deleteServer: (id: string) => apiInstance.delete(`/servers/${id}`),
    startServer: (id: string) => apiInstance.post(`/servers/${id}/start`),
    stopServer: (id: string) => apiInstance.post(`/servers/${id}/stop`),
    killServer: (id: string) => apiInstance.post(`/servers/${id}/kill`),
    getFiles: (id: string) => apiInstance.get(`/servers/${id}/files`),
    getFileContent: (id: string, path: string) => apiInstance.get(`/servers/${id}/file`, { params: { path } }),
    updateFileContent: (id: string, path: string, content: string) => apiInstance.put(`/servers/${id}/file`, { path, content }),
    getPortRange: () => apiInstance.get('/settings/port-range'),
    updatePortRange: (data: { start: number, end: number }) => apiInstance.put('/settings/port-range', data),
};
