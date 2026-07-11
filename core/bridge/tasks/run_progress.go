package tasks

type RunProgressWriter interface {
	WriteRunningRunLog(update RunningRunLogUpdate) error
}

type RunningRunLogUpdate struct {
	RunCards []RunCard
}

type runningRunLogWriter struct {
	store *Store
	base  RunLog
}

func newRunningRunLogWriter(store *Store, base RunLog) RunProgressWriter {
	if store == nil {
		return nil
	}
	return runningRunLogWriter{
		store: store,
		base:  base,
	}
}

func (w runningRunLogWriter) WriteRunningRunLog(update RunningRunLogUpdate) error {
	log := w.base
	log.RunCards = CloneRunCards(update.RunCards)
	return w.store.AppendRunLog(log)
}
