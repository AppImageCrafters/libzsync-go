package zsync

type ProgressReporter interface {
	SetDescription(label string)
	SetProgress(value int64)
	SetTotal(value int64)

	Write(p []byte) (n int, err error)
}

type dummyProgressReporter struct{}

func NewDummyProgressReporter() ProgressReporter {
	return &dummyProgressReporter{}
}

func (d dummyProgressReporter) SetDescription(label string) {
}

func (d dummyProgressReporter) SetProgress(value int64) {
}

func (d dummyProgressReporter) SetTotal(value int64) {
}

func (d dummyProgressReporter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
