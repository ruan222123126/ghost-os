package memory

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Evolver 定义记忆演化能力（摘要、压缩、关联）。
type Evolver interface {
	// Summarize 对消息列表生成摘要。
	Summarize(ctx context.Context, entries []MemoryEntry) (string, error)

	// ExtractEntities 从内容中提取关键实体（人名、项目名、技术栈等）。
	ExtractEntities(ctx context.Context, content string) ([]string, error)

	// ComputeImportance 计算记忆条目的重要度（0.0-1.0）。
	ComputeImportance(ctx context.Context, entry MemoryEntry) (float64, error)
}

// EvolutionConfig 定义后台演化任务配置。
type EvolutionConfig struct {
	Interval       time.Duration // 演化周期（默认 1 小时）
	WarmThreshold  int           // L2 条目数超过此值触发压缩
	ColdAgeLimit   time.Duration // L3 超过此时长的条目可被总结归档
	EnableAutoRun  bool          // 是否自动启动后台协程
}

// EvolutionTask 后台演化任务，负责记忆的自动整理和优化。
type EvolutionTask struct {
	manager *MemoryManager
	evolver Evolver
	config  EvolutionConfig
	stopCh  chan struct{}
}

// NewEvolutionTask 创建演化任务实例。
func NewEvolutionTask(manager *MemoryManager, evolver Evolver, config EvolutionConfig) *EvolutionTask {
	if config.Interval <= 0 {
		config.Interval = 1 * time.Hour
	}
	if config.WarmThreshold <= 0 {
		config.WarmThreshold = 80 // 默认 80% 容量触发
	}
	if config.ColdAgeLimit <= 0 {
		config.ColdAgeLimit = 7 * 24 * time.Hour // 默认 7 天
	}

	return &EvolutionTask{
		manager: manager,
		evolver: evolver,
		config:  config,
		stopCh:  make(chan struct{}),
	}
}

// Start 启动后台演化协程（类似人类睡眠时的记忆整理）。
func (t *EvolutionTask) Start() {
	if !t.config.EnableAutoRun {
		return
	}
run()
	log.Printf("[Memory] Evolution task started, interval=%v", t.config.Interval)
}

// Stop 停止后台演化任务。
func (t *EvolutionTask) Stop() {
	close(t.stopCh)
}

// RunOnce 手动触发一次演化（用于测试或主动整理）。
func (t *EvolutionTask) RunOnce(ctx context.Context) error {
	return t.evolve(ctx)
}

func (t *EvolutionTask) run() {
	ticker := time.NewTicker(t.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			if err := t.evolve(ctx); err != nil {
				log.Printf("[Memory] Evolution error: %v", err)
			}
			cancel()
		case <-t.stopCh:
			return
		}
	}
}

func (t *EvolutionTask) evolve(ctx context.Context) error {
	// 1. 压缩 L2 温数据：当容量接近上限时，将低频条目降级到 L3
	if err := t.compressWarmMemory(ctx); err != nil {
		return fmt.Errorf("compress warm memory: %w", err)
	}

	// 2. 总结 L3 冷数据：为旧会话生成摘要和实体标签
	if err := t.summarizeColdMemory(ctx); err != nil {
		return fmt.Errorf("summarize cold memory: %w", err)
	}

	// 3. 清理过期数据：删除超过保留期限的低价值记忆
	if err := t.pruneExpiredMemory(ctx); err != nil {
		return fmt.Errorf("prune expired memory: %w", err)
	}

	return nil
}

// compressWarmMemory 将 L2 中低频访问的条目迁移到 L3。
func (t *EvolutionTask) compressWarmMemory(ctx context.Context) error {
	if t.evolver == nil {
		return nil
	}

	// 查询 L2 中所有条目
	entries, err := t.manager.warm.Retrieve(MemoryQuery{Limit: 0})
	if err != nil {
		return err
	}

	capacity := t.manager.warm.capacity
	threshold := (capacity * t.config.WarmThreshold) / 100

	if len(entries) < threshold {
		return nil // 容量充足，无需压缩
	}

	// 按访问频率排序，将低频条目归档
	lowFreqCount := len(entries) - threshold
	if lowFreqCount <= 0 {
		return nil
	}

	// 这里简化处理：将最旧的条目归档
	// 实际可以根据 AccessCount 和 DecayFactor 综合判断
	for i := 0; i < lowFreqCount && i < len(entries); i++ {
		entry := entries[i]
		sessionID, ok := entry.Metadata["session_id"].(string)
		if !ok || sessionID == "" {
			continue
		}

		// 归档到 L3（这里需要从 session store 获取完整消息）
		if err := t.manager.ArchiveToCold(sessionID); err != nil {
			log.Printf("[Memory] Failed to archive session %s: %v", sessionID, err)
			continue
		}

		// 从 L2 删除
		if err := t.manager.warm.Delete(entry.ID); err != nil {
			log.Printf("[Memory] Failed to delete warm entry %s: %v", entry.ID, err)
		}
	}

	return nil
}

// summarizeColdMemory 为 L3 中的旧会话生成摘要和实体标签。
func (t *EvolutionTask) summarizeColdMemory(ctx context.Context) error {
	if t.evolver == nil {
		return nil
	}

	// 查询需要总结的会话（超过 ColdAgeLimit 且未总结的）
	cutoff := time.Now().UTC().Add(-t.config.ColdAgeLimit)
	timeRange := TimeRange{End: cutoff}

	sessionIDs, err := t.manager.cold.ListSessions(timeRange)
	if err != nil {
		return err
	}

	for _, sessionID := range sessionIDs {
		// 检查是否已有摘要
		query := MemoryQuery{
			Metadata: map[string]any{
				"session_id": sessionID,
				"layer":      "cold",
			},
			Limit: 1,
		}
		entries, err := t.manager.cold.Retrieve(query)
		if err != nil || len(entries) == 0 {
			continue
		}

		// 如果已有摘要，跳过
		if entries[0].Type == MemoryTypeSummary {
			continue
		}

		// 生成摘要
		summary, err := t.evolver.Summarize(ctx, entries)
		if err != nil {
			log.Printf("[Memory] Failed to summarize session %s: %v", sessionID, err)
			continue
		}

		// 提取实体
		entities, err := t.evolver.ExtractEntities(ctx, summary)
		if err != nil {
			log.Printf("[Memory] Failed to extract entities for session %s: %v", sessionID, err)
			entities = []string{}
		}

		// 更新归档文件（添加摘要和实体）
		if err := t.manager.cold.UpdateSummary(sessionID, summary, entities); err != nil {
			log.Printf("[Memory] Failed to update summary for session %s: %v", sessionID, err)
		}
	}

	return nil
}

// pruneExpiredMemory 清理过期的低价值记忆。
func (t *EvolutionTask) pruneExpiredMemory(ctx context.Context) error {
	// 这里可以实现更复杂的清理策略
	// 例如：删除超过 1 年且访问次数为 0 的记忆
	// 当前版本保持简单，依赖 L2 的 TTL 自动清理
	return nil
}
