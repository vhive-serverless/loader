package common

const (
	// 1ms (min. billing unit of AWS)
	MIN_EXEC_TIME_MILLI = 1

	// 60s (avg. p96 from Wild)
	MAX_EXEC_TIME_MILLI = 60e3
)

const (
	// Ten-minute warmup for unifying the starting time when the experiments consists of multiple runs.
	WARMUP_DURATION_MINUTES = 10

	// https://docs.aws.amazon.com/lambda/latest/dg/configuration-function-common.html#configuration-memory-console
	MAX_MEM_QUOTA_MIB = 10_240
	MIN_MEM_QUOTA_MIB = 128

	// Machine overcommitment ratio to provide to CPU requests in YAML specification
	OVERCOMMITMENT_RATIO = 10
)

type IatDistribution int

const (
	Exponential IatDistribution = iota
	Uniform     IatDistribution = iota
	Equidistant IatDistribution = iota
)

const (
	OneSecondInMicroseconds = 1_000_000.0
)

type ExperimentPhase int

const (
	WarmupPhase    ExperimentPhase = 1
	ExecutionPhase                 = 2
)
