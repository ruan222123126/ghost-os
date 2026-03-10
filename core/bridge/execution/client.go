package execution

import (
	"context"
	"strings"
)

// Client 定义 bridge 调用 execution layer 的最小接口。
type Client interface {
	Call(ctx context.Context, action string, params map[string]any, traceID string) (map[string]any, error)
}

// Closeable 表示底层持有可主动释放资源的 client。
type Closeable interface {
	Close() error
}

// ClientOptions 描述 execution client 的装配策略。
type ClientOptions struct {
	Persistent bool
	// NativeBinaryPath 为 native 二进制的显式路径（优先级最高）。
	NativeBinaryPath string
	// NativeBinaryRoots 为搜索 native 二进制时追加的根目录。
	NativeBinaryRoots []string
	// NativeBinaryCandidates 为搜索 native 二进制时使用的候选相对路径。
	NativeBinaryCandidates []string
}

func NewClientWithOptions(opts ClientOptions) Client {
	locator := nativeBinaryLocator{
		configuredPath: strings.TrimSpace(opts.NativeBinaryPath),
		roots:          append([]string(nil), opts.NativeBinaryRoots...),
		candidates:     append([]string(nil), opts.NativeBinaryCandidates...),
	}
	if opts.Persistent {
		return newPersistentNativeClientWithLocator(locator)
	}
	return newNativeClientWithLocator(locator)
}

func CloseClient(client Client) error {
	if closer, ok := client.(Closeable); ok {
		return closer.Close()
	}
	return nil
}
