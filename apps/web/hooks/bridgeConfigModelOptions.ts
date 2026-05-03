import type { BridgeConfig, ProviderConfig, ProviderModelOption } from '@/lib/types';

export function buildProviderModelOptions(
  config: BridgeConfig | null,
  providers: ProviderConfig[],
): ProviderModelOption[] {
  const options: ProviderModelOption[] = [];
  const seen = new Set<string>();
  const currentProvider = config?.provider ?? '';
  const activeProviders = providers.filter((provider) => {
    return stringsEqualIgnoreCase(provider.name, currentProvider);
  });

  const pushOption = (provider: ProviderConfig | null, model: string) => {
    const trimmedModel = model.trim();
    if (!trimmedModel) {
      return;
    }

    const providerName = provider?.name ?? config?.provider ?? 'runtime';
    const key = `${providerName.toLowerCase()}::${trimmedModel.toLowerCase()}`;
    if (seen.has(key)) {
      return;
    }

    seen.add(key);
    options.push({
      providerName,
      providerType: provider?.type ?? config?.provider_type ?? 'custom',
      model: trimmedModel,
    });
  };

  for (const provider of activeProviders) {
    for (const model of provider.models ?? []) {
      pushOption(provider, model);
    }

    // Active runtime model must stay selectable even if the provider card has no explicit model list yet.
    if (config) {
      pushOption(provider, config.model);
    }
  }

  if (options.length === 0 && config?.model) {
    pushOption(null, config.model);
  }

  return options;
}

export function resolveActiveModelOption(
  config: BridgeConfig | null,
  modelOptions: ProviderModelOption[],
): ProviderModelOption | null {
  if (!config) {
    return null;
  }

  return modelOptions.find((option) => {
    return stringsEqualIgnoreCase(option.providerName, config.provider)
      && stringsEqualIgnoreCase(option.model, config.model);
  }) ?? {
    providerName: config.provider,
    providerType: config.provider_type,
    model: config.model,
  };
}

export function stringsEqualIgnoreCase(left: string, right: string): boolean {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}
