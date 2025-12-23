export interface Server {
    id: string;
    name: string;
    version: string;
    loader: string;
    port: number;
    ram: number;
    status: "STOPPED" | "RUNNING" | "STARTING" | "STOPPING" | "CREATING";
    progress?: number;
    progressMessage?: string;
}

export interface Backup {
    name: string;
    size: number;
}
