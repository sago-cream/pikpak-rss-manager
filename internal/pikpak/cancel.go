package pikpak

import "context"

type TaskCanceller interface {
	CancelTask(context.Context, string) error
}

func (c *Client) CancelTask(ctx context.Context, id string) error {
	if !c.canCancel {
		return &Error{Kind: "permanent", Message: "官方 MCP 缺少必要工具 task_rm"}
	}
	// Remove the task only. Never delete associated downloaded files.
	return c.call(ctx, "task_rm", map[string]any{"ids": []string{id}, "delete-files": false}, nil)
}
