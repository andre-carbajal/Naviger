export interface Server {
    id: string;
    name: string;
    version: string;
    loader: string;
    port: number;
    ram: number;
    status: "STOPPED" | "RUNNING" | "STARTING" | "STOPPING" | "CREATING";
    customArgs?: string;
    progress?: number;
    progressMessage?: string;
    steps?: ProgressStep[];
}

export interface ProgressStep {
    label: string;
    state: 'pending' | 'running' | 'done' | 'failed';
    progress?: number;
}

export interface Backup {
    name: string;
    size: number;
    status?: 'CREATING' | 'READY' | 'ERROR';
    progress?: number;
    requestId?: string;
    progressMessage?: string;
}

export interface ServerStats {
    cpu: number;
    ram: number;
    disk: number;
}
