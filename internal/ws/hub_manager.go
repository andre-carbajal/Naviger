package ws

import "sync"

type HubManager struct {
	hubs map[string]*Hub
	mu   sync.Mutex
}

func NewHubManager() *HubManager {
	return &HubManager{
		hubs: make(map[string]*Hub),
	}
}

func (m *HubManager) GetHub(serverID string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[serverID]; ok {
		return hub
	}

	hub := NewHub()
	go hub.Run()
	m.hubs[serverID] = hub
	return hub
}

func (m *HubManager) RemoveHub(serverID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[serverID]; ok {
		hub.Stop()
		delete(m.hubs, serverID)
	}
}
