package traceroute

// Status describes the lifecycle state of one traceroute request.
type Status string

const (
	StatusStarted   Status = "started"
	StatusProgress  Status = "progress"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusTimedOut  Status = "timed_out"
)
