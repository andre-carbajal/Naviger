import React, {useEffect, useState} from 'react';
import {api} from '../services/api';

const Settings: React.FC = () => {
    const [portRange, setPortRange] = useState({start: 0, end: 0});
    const [initialPortRange, setInitialPortRange] = useState({start: 0, end: 0});
    const [loading, setLoading] = useState(true);
    const [isSaving, setIsSaving] = useState(false);
    const [hasChanges, setHasChanges] = useState(false);

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

    if (loading) return <div>Loading settings...</div>;

    return (
        <div className="settings-page">
            <h1 style={{marginBottom: '30px'}}>Settings</h1>

            <div className="card" style={{maxWidth: '600px'}}>
                <h2 style={{fontSize: '1.2rem', marginTop: 0}}>Network Configuration</h2>
                <p style={{color: 'var(--text-muted)', marginBottom: '20px'}}>
                    Define the range of ports that the manager can assign to new servers.
                </p>

                <div style={{display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px'}}>
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

                <div style={{marginTop: '20px', display: 'flex', justifyContent: 'flex-end'}}>
                    <button
                        className="btn btn-primary"
                        onClick={handleSave}
                        disabled={!hasChanges || isSaving}
                    >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                    </button>
                </div>
            </div>
        </div>
    );
};

export default Settings;
