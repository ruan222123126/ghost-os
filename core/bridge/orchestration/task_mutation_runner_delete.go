package orchestration

func (r taskMutationRunner) Delete(params taskIDParams) (taskDeleteResponse, error) {
	return r.inner().Delete(params)
}
