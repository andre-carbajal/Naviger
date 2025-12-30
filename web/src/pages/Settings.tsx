import React, {useEffect, useState} from 'react';
import {api} from '../services/api';
import {Button} from '../components/ui/Button';

const Settings: React.FC = () => {
    const [portRange, setPortRange] = useState({start: 0, end: 0});
    const [initialPortRange, setInitialPortRange] = useState({start: 0, end: 0});
    const [loading, setLoading] = useState(true);
    const [isSaving, setIsSaving] = useState(false);
    const [hasChanges, setHasChanges] = useState(false);
    const [isRestarting, setIsRestarting] = useState(false);

    useEffect(() => {
        const fetchSettings = async () => {
            try {
                const res = await api.getPortRange();
                setPortRange(res.data);
                setInitialPortRange(res.data);
            } catch (err) {
                console.error("Failed to fetch settings:", err);
            } finally {
                setLoading(false);
            }
        };
        fetchSettings();
    }, []);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const {name, value} = e.target;
        const newPortRange = {...portRange, [name]: parseInt(value, 10)};
        setPortRange(newPortRange);
        setHasChanges(JSON.stringify(newPortRange) !== JSON.stringify(initialPortRange));
    };

    const handleSave = async () => {
        setIsSaving(true);
        try {
            await api.updatePortRange(portRange);
            setInitialPortRange(portRange);
            setHasChanges(false);
        } catch (err) {
            console.error("Failed to save settings:", err);
        } finally {
            setIsSaving(false);
        }
    };

    const handleRestart = async () => {
        if (!confirm("Are you sure you want to restart the daemon? This will stop all running servers.")) {
            return;
        }
        setIsRestarting(true);
        try {
            await api.restartDaemon();
            alert("Daemon restart command sent. The page may lose connection briefly.");
        } catch (err: any) {
            if (err.code === "ERR_NETWORK" || err.message === "Network Error") {
                 alert("Daemon restart command sent. The page may lose connection briefly.");
            } else {
                console.error("Failed to restart daemon:", err);
                alert("Failed to restart daemon.");
            }
        } finally {
            setIsRestarting(false);
        }
    };

    if (loading) return <div>Loading settings...</div>;

    return (
        <div className="settings-page">
            <h1>Settings</h1>

            <div className="card">
                <h2>Network Configuration</h2>
                <p>
                    Define the range of ports that the manager can assign to new servers.
                </p>

                <div>
                    <div className="form-group">
                        <label>Start Port</label>
                        <input
                            type="number"
                            name="start"
                            className="form-input"
                            value={portRange.start}
                            onChange={handleChange}
                        />
                    </div>
                    <div className="form-group">
                        <label>End Port</label>
                        <input
                            type="number"
                            name="end"
                            className="form-input"
                            value={portRange.end}
                            onChange={handleChange}
                        />
                    </div>
                </div>

                <div>
                    <Button
                        onClick={handleSave}
                        disabled={!hasChanges || isSaving}
                    >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                    </Button>
                </div>
            </div>

            <div className="card" style={{marginTop: '20px'}}>
                <h2>System</h2>
                <p>
                    Manage the Naviger Daemon process.
                </p>
                <div>
                    <Button
                        variant="danger"
                        onClick={handleRestart}
                        disabled={isRestarting}
                    >
                        {isRestarting ? 'Restarting...' : 'Restart Daemon'}
                    </Button>
                </div>
            </div>
        </div>
    );
};

export default Settings;
