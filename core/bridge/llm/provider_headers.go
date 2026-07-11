package llm

func (c *Client) providerHeaders(defaults map[string]string) map[string]string {
	if len(defaults) == 0 && len(c.opts.Headers) == 0 {
		return nil
	}

	headers := make(map[string]string, len(defaults)+len(c.opts.Headers))
	mergeStringHeaders(headers, defaults)
	mergeStringHeaders(headers, c.opts.Headers)
	if len(headers) == 0 {
		return nil
	}
	return headers
}
