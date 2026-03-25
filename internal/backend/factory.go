package backend

import (
	"fmt"

	"github.com/llm-proxy/internal/config"
)

type Factory struct {
	backends map[string]Backend
}

func NewFactory(cfg map[string]config.Backend, apiKeys map[string]string) (*Factory, error) {
	backends := make(map[string]Backend)

	for name, backendCfg := range cfg {
		apiKey := resolveAPIKey(backendCfg.APIKey, apiKeys)

		switch backendCfg.Type {
		case "openrouter", "openai", "openai-compatible":
			backends[name] = NewOpenRouterBackend(
				name,
				backendCfg.BaseURL,
				apiKey,
				backendCfg.ExtraHeaders,
			)
		case "anthropic":
			backends[name] = NewOpenRouterBackend(
				name,
				backendCfg.BaseURL,
				apiKey,
				backendCfg.ExtraHeaders,
			)
		default:
			return nil, fmt.Errorf("unsupported backend type: %s", backendCfg.Type)
		}
	}

	return &Factory{backends: backends}, nil
}

func (f *Factory) Get(name string) (Backend, error) {
	backend, ok := f.backends[name]
	if !ok {
		return nil, fmt.Errorf("backend not found: %s", name)
	}
	return backend, nil
}

func (f *Factory) List() []string {
	names := make([]string, 0, len(f.backends))
	for name := range f.backends {
		names = append(names, name)
	}
	return names
}

func resolveAPIKey(key string, apiKeys map[string]string) string {
	if key == "" {
		return ""
	}

	if key[0] == '$' && key[1] == '{' && key[len(key)-1] == '}' {
		refKey := key[2 : len(key)-1]
		if val, ok := apiKeys[refKey]; ok {
			return val
		}
	}

	return key
}
