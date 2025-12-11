export interface Server {
    id: string;
    name: string;
    version: string;
    loader: string;
    port: number;
    ram: number;
    status: "STOPPED" | "RUNNING" | "STARTING";
}