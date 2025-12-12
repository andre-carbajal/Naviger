import {useEffect, useState} from 'react';
import {BrowserRouter, Route, Routes} from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import ServerDetail from './pages/ServerDetail';
import Settings from './pages/Settings';
import Backups from './pages/Backups';

const App = () => {
    const [notification, setNotification] = useState<{ message: string; visible: boolean }>({
        message: '',
        visible: false
    });

    useEffect(() => {
        const handleNetworkError = (event: Event) => {
            const customEvent = event as CustomEvent;
            setNotification({message: customEvent.detail.message, visible: true});
        };

        window.addEventListener('network-error', handleNetworkError);

        return () => {
            window.removeEventListener('network-error', handleNetworkError);
        };
    }, []);

    useEffect(() => {
        if (notification.visible) {
            const timer = setTimeout(() => {
                setNotification({...notification, visible: false});
            }, 5000);
            return () => clearTimeout(timer);
        }
    }, [notification]);

    return (
        <BrowserRouter>
            {notification.visible && (
                <div className="notification">
                    {notification.message}
                </div>
            )}
            <Routes>
                <Route path="/" element={<Layout/>}>
                    <Route index element={<Dashboard/>}/>
                    <Route path="servers/backups/all" element={<Backups/>}/>
                    <Route path="servers/:id" element={<ServerDetail/>}/>
                    <Route path="servers/:id/backups" element={<Backups/>}/>
                    <Route path="settings" element={<Settings/>}/>
                </Route>
            </Routes>
        </BrowserRouter>
    );
};

const styleSheet = document.createElement("style");
styleSheet.type = "text/css";
styleSheet.innerText = `
  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-20px); }
    to { opacity: 1; transform: translateY(0); }
  }
  @keyframes fadeOut {
    from { opacity: 1; transform: translateY(0); }
    to { opacity: 0; transform: translateY(-20px); }
  }
`;
document.head.appendChild(styleSheet);

export default App;
