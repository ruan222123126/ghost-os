package tasks

import (
	"fmt"
	"time"

	cron "github.com/robfig/cron/v3"
)

type taskSchedulePlan struct {
	kind     string
	interval time.Duration
	cronExpr string
	cron     cron.Schedule
}

func buildTaskSchedulePlan(task ScheduledTask) (taskSchedulePlan, error) {
	switch task.ScheduleType {
	case ScheduleTypeInterval:
		if task.IntervalSeconds <= 0 {
			return taskSchedulePlan{}, fmt.Errorf("%w: interval_seconds must be > 0", ErrInvalidTaskConfig)
		}
		return taskSchedulePlan{
			kind:     ScheduleTypeInterval,
			interval: time.Duration(task.IntervalSeconds) * time.Second,
		}, nil
	case ScheduleTypeCron:
		schedule, err := cron.ParseStandard(task.CronExpr)
		if err != nil {
			return taskSchedulePlan{}, fmt.Errorf("%w: invalid cron_expr: %w", ErrInvalidTaskConfig, err)
		}
		return taskSchedulePlan{
			kind:     ScheduleTypeCron,
			cronExpr: task.CronExpr,
			cron:     schedule,
		}, nil
	default:
		return taskSchedulePlan{}, fmt.Errorf(
			"%w: unsupported schedule type %q",
			ErrInvalidTaskConfig,
			task.ScheduleType,
		)
	}
}

func (p taskSchedulePlan) initialNext(now time.Time, task ScheduledTask) time.Time {
	if !task.NextRunAt.IsZero() && task.NextRunAt.After(now) {
		return task.NextRunAt.UTC()
	}
	switch p.kind {
	case ScheduleTypeInterval:
		return now.UTC().Add(p.interval)
	case ScheduleTypeCron:
		return p.cron.Next(now.UTC()).UTC()
	default:
		return now.UTC()
	}
}

func (p taskSchedulePlan) nextAfter(previous time.Time, now time.Time) time.Time {
	switch p.kind {
	case ScheduleTypeInterval:
		next := previous.UTC().Add(p.interval)
		if next.After(now.UTC()) {
			return next
		}
		return now.UTC().Add(p.interval)
	case ScheduleTypeCron:
		next := p.cron.Next(previous.UTC()).UTC()
		if next.After(now.UTC()) {
			return next
		}
		return p.cron.Next(now.UTC()).UTC()
	default:
		return now.UTC()
	}
}

func nextTaskRunAt(task ScheduledTask, now time.Time) (time.Time, error) {
	plan, err := buildTaskSchedulePlan(task)
	if err != nil {
		return time.Time{}, err
	}
	return plan.initialNext(now.UTC(), task), nil
}
