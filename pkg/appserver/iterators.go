package appserver

import "context"

func (c *Client) ListAllThreads(ctx context.Context, params ThreadListParams, yield func(Thread) bool) error {
	for {
		result, err := c.ListThreads(ctx, params)
		if err != nil {
			return err
		}

		for _, thread := range result.Data {
			if !yield(thread) {
				return nil
			}
		}

		if result.NextCursor == nil || *result.NextCursor == "" {
			return nil
		}
		params.Cursor = result.NextCursor
	}
}

func (c *Client) ListAllModels(ctx context.Context, params ModelListParams, yield func(ModelInfo) bool) error {
	for {
		result, err := c.ListModels(ctx, params)
		if err != nil {
			return err
		}

		for _, model := range result.Data {
			if !yield(model) {
				return nil
			}
		}

		if result.NextCursor == nil || *result.NextCursor == "" {
			return nil
		}
		params.Cursor = result.NextCursor
	}
}
