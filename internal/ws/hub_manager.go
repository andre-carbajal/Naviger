package ws

import "sync"

type HubManager struct {
	hubs               map[string]*Hub
	mu                 sync.Mutex
	defaultHistorySize int
}

func NewHubManager(defaultHistorySize int) *HubManager {
	return &HubManager{
		hubs:               make(map[string]*Hub),
		defaultHistorySize: defaultHistorySize,
	}
}

func (m *HubManager) GetHub(serverID string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[serverID]; ok {
		return hub
	}

	hub := NewHubWithHistorySize(m.defaultHistorySize)
	go hub.Run()
	m.hubs[serverID] = hub
	return hub
}

func (m *HubManager) RemoveHub(serverID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[serverID]; ok {
		hub.ClearLogs()
		hub.Stop()
		delete(m.hubs, serverID)
	}
}

func (m *HubManager) SetDefaultHistorySize(size int) {
	if size < 0 {
		size = 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultHistorySize = size
	for _, hub := range m.hubs {
		if hub != nil {
			hub.SetHistorySize(size)
		}
	}
}
