package sdk

import "fmt"

func (c *Client) ListAllBackups() ([]BackupInfo, error) {
	var backups []BackupInfo
	err := c.get("/backups", &backups)
	return backups, err
}

func (c *Client) ListServerBackups(serverID string) ([]BackupInfo, error) {
	var backups []BackupInfo
	err := c.get(fmt.Sprintf("/servers/%s/backups", serverID), &backups)
	return backups, err
}

func (c *Client) CreateBackup(serverID, name string) (*struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}, error) {
	payload := map[string]string{
		"name": name,
	}
	var result struct {
		Message string `json:"message"`
		Path    string `json:"path"`
	}
	err := c.post(fmt.Sprintf("/servers/%s/backup", serverID), payload, &result)
	return &result, err
}

func (c *Client) DeleteBackup(name string) error {
	return c.delete(fmt.Sprintf("/backups/%s", name))
}

func (c *Client) RestoreBackup(backupName string, req RestoreBackupRequest) error {
	return c.post(fmt.Sprintf("/backups/%s/restore", backupName), req, nil)
}
