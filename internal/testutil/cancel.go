package testutil

import "context"

func (c *Cloud) CancelTask(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.boundary("cancel"); err != nil {
		return err
	}
	delete(c.Tasks, id)
	return nil
}
