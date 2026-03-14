package config

func materializeRuntimeGraphQLIfNeeded(
	fileCfg *bridgeFileConfig,
	current runtimeConfig,
	req configUpdateRequest,
) {
	if fileCfg == nil || !requiresRuntimeGraphQLMaterialization(req) {
		return
	}
	if fileCfg.GraphQLEnabled == nil {
		fileCfg.GraphQLEnabled = boolPointer(current.GraphQL.Enabled)
	}
	if fileCfg.GraphQLEndpoint == nil {
		fileCfg.GraphQLEndpoint = optionalStringPointer(current.GraphQL.Endpoint)
	}
	if fileCfg.GraphQLAPIKey == nil {
		fileCfg.GraphQLAPIKey = optionalStringPointer(current.GraphQL.APIKey)
	}
	if fileCfg.GraphQLSchemaPath == nil {
		fileCfg.GraphQLSchemaPath = optionalStringPointer(current.GraphQL.SchemaPath)
	}
	if fileCfg.GraphQLTimeoutMS == nil {
		fileCfg.GraphQLTimeoutMS = optionalIntPointer(current.GraphQL.TimeoutMS)
	}
	if fileCfg.GraphQLMaxResponseBytes == nil {
		fileCfg.GraphQLMaxResponseBytes = optionalIntPointer(current.GraphQL.MaxResponseBytes)
	}
	if fileCfg.GraphQLHeaders == nil && current.GraphQL.Headers != nil {
		fileCfg.GraphQLHeaders = cloneStringMap(current.GraphQL.Headers)
	}
}

func requiresRuntimeGraphQLMaterialization(req configUpdateRequest) bool {
	return req.GraphQLEnabled != nil ||
		req.GraphQLEndpoint != nil ||
		req.GraphQLAPIKey != nil ||
		req.GraphQLSchemaPath != nil ||
		req.GraphQLTimeoutMS != nil ||
		req.GraphQLMaxResponseBytes != nil ||
		req.GraphQLHeaders != nil
}

func applyGraphQLRuntimeFields(fileCfg *bridgeFileConfig, req configUpdateRequest) {
	if req.GraphQLEnabled != nil {
		fileCfg.GraphQLEnabled = boolPointer(*req.GraphQLEnabled)
	}
	if req.GraphQLEndpoint != nil {
		fileCfg.GraphQLEndpoint = cloneOptionalStringPointer(req.GraphQLEndpoint)
	}
	if req.GraphQLAPIKey != nil {
		fileCfg.GraphQLAPIKey = cloneOptionalStringPointer(req.GraphQLAPIKey)
	}
	if req.GraphQLSchemaPath != nil {
		fileCfg.GraphQLSchemaPath = cloneOptionalStringPointer(req.GraphQLSchemaPath)
	}
	if req.GraphQLTimeoutMS != nil {
		fileCfg.GraphQLTimeoutMS = optionalIntPointer(*req.GraphQLTimeoutMS)
	}
	if req.GraphQLMaxResponseBytes != nil {
		fileCfg.GraphQLMaxResponseBytes = optionalIntPointer(*req.GraphQLMaxResponseBytes)
	}
	if req.GraphQLHeaders != nil {
		fileCfg.GraphQLHeaders = cloneStringMap(req.GraphQLHeaders)
	}
}

func cloneStringMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func optionalIntPointer(value int) *int {
	v := value
	return &v
}

func boolPointer(value bool) *bool {
	v := value
	return &v
}
