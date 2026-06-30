import type { TaskScheduleType } from "../mobileTypes";

interface ScheduledTaskLike {
  cron_expr?: string;
  interval_seconds?: number;
  schedule_type: TaskScheduleType;
}

export function formatSchedule(task: ScheduledTaskLike): string {
  if (task.schedule_type === "interval") {
    return `每 ${task.interval_seconds ?? 0} 秒`;
  }
  return `Cron ${task.cron_expr ?? ""}`;
}
