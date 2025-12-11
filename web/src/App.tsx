import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import ServerDetail from './pages/ServerDetail';
import Settings from './pages/Settings';

const App = () => {
  const [notification, setNotification] = useState<{ message: string; visible: boolean }>({ message: '', visible: false });

  useEffect(() => {
    const handleNetworkError = (event: Event) => {
      const customEvent = event as CustomEvent;
      setNotification({ message: customEvent.detail.message, visible: true });
    };

    window.addEventListener('network-error', handleNetworkError);

    return () => {
      window.removeEventListener('network-error', handleNetworkError);
    };
  }, []);

  useEffect(() => {
    if (notification.visible) {
      const timer = setTimeout(() => {
        setNotification({ ...notification, visible: false });
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [notification]);

  return (
    <BrowserRouter>
      {notification.visible && (
        <div style={notificationStyles}>
          {notification.message}
        </div>
      )}
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="servers" element={<Navigate to="/" replace />} />
          <Route path="servers/:id" element={<ServerDetail />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
};

const notificationStyles: React.CSSProperties = {
  position: 'fixed',
  top: '20px',
  right: '20px',
  backgroundColor: 'var(--danger-color, #f44336)',
  color: 'white',
  padding: '15px 20px',
  borderRadius: '8px',
  zIndex: 1000,
  boxShadow: '0 4px 8px rgba(0,0,0,0.2)',
  animation: 'fadeIn 0.5s, fadeOut 0.5s 4.5s'
};

// Add keyframes for animations
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
