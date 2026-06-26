import type { BridgeConfig, ProviderConfig, ProviderModelOption } from '@/lib/types';

interface ProviderModelSource {
  providerName: string;
  providerType: ProviderModelOption['providerType'];
}

export function buildProviderModelOptions(
  config: BridgeConfig | null,
  providers: ProviderConfig[],
): ProviderModelOption[] {
  const collector = createProviderModelCollector();
  const activeProviders = activeProvidersForConfig(config, providers);

  for (const provider of activeProviders) {
    const source = providerModelSource(provider);
    for (const model of providerModels(provider)) {
      collector.add(source, model);
    }

    // Active runtime model must stay selectable even if the provider card has no explicit model list yet.
    if (config) {
      collector.add(source, config.model);
    }
  }

  if (collector.isEmpty() && config) {
    collector.add(runtimeModelSource(config), config.model);
  }

  return collector.values();
}

function createProviderModelCollector() {
  const options: ProviderModelOption[] = [];
  const seen = new Set<string>();

  function add(source: ProviderModelSource, model: string): void {
    const trimmedModel = model.trim();
    if (!trimmedModel) {
      return;
    }

    const key = modelOptionKey(source.providerName, trimmedModel);
    if (seen.has(key)) {
      return;
    }

    seen.add(key);
    options.push({
      providerName: source.providerName,
      providerType: source.providerType,
      model: trimmedModel,
    });
  }

  return {
    add,
    isEmpty: () => options.length === 0,
    values: () => options,
  };
}

function activeProvidersForConfig(config: BridgeConfig | null, providers: ProviderConfig[]): ProviderConfig[] {
  const currentProvider = config ? config.provider : '';
  return providers.filter((provider) => stringsEqualIgnoreCase(provider.name, currentProvider));
}

function providerModels(provider: ProviderConfig): string[] {
  if (!provider.models) {
    return [];
  }
  return provider.models;
}

function providerModelSource(provider: ProviderConfig): ProviderModelSource {
  return {
    providerName: provider.name,
    providerType: provider.type,
  };
}

function runtimeModelSource(config: BridgeConfig): ProviderModelSource {
  return {
    providerName: config.provider,
    providerType: config.provider_type,
  };
}

function modelOptionKey(providerName: string, model: string): string {
  return `${providerName.toLowerCase()}::${model.toLowerCase()}`;
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
