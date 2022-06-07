package generate

import (
	"math/rand"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	util "github.com/eth-easl/loader/pkg"
	tc "github.com/eth-easl/loader/pkg/trace"
)

const (
	STATIONARY_P_VALUE    = 0.05
	TRACE_WARMUP_DURATION = 6 // Six-minute warmup for unifying the starting time.

	OVERFLOAD_THRESHOLD = 0.3
	OVERFLOAD_TOLERANCE = 2

	FORCE_TIMEOUT_MINUTE = 15 // 15 min maximum waiting time based upon max. function duration of popular clouds.

	RPS_WARMUP_FRACTION  = 0.5
	MAX_RPS_STARTUP_STEP = 5

	MAX_EXEC_TIME_MILLI = 10e3 // 10s (avg. p90 from Wild).
	MIN_EXEC_TIME_MILLI = 1    // 1ms (min. billing unit of AWS).
	MAX_MEM_MIB         = 200  // 170MiB (max. p50 for the whole App from Wild).
	MIN_MEM_MIB         = 1
)

/** Seed the math/rand package for it to be different on each run. */
// func init() {
// 	rand.Seed(time.Now().UnixNano())
// }

var iatRand *rand.Rand
var invRand *rand.Rand
var specRand *rand.Rand

func InitSeed(s int64) {
	iatRand = rand.New(rand.NewSource(s))
	invRand = rand.New(rand.NewSource(s))
	specRand = rand.New(rand.NewSource(s))
}

func GenerateInterarrivalTimesInMicro(invocationsPerMinute int, uniform bool) []float64 {
	oneSecondInMicro := 1000_000.0
	oneMinuteInMicro := 60*oneSecondInMicro - 1000

	rps := float64(invocationsPerMinute) / 60
	interArrivalTimes := []float64{}

	totoalDuration := 0.0
	for i := 0; i < invocationsPerMinute; i++ {
		var iat float64
		if uniform {
			iat = oneSecondInMicro / rps
		} else {
			iat = iatRand.ExpFloat64() / rps * oneSecondInMicro
		}
		//* Only guarantee microsecond-level ganularity.
		if iat < 1 {
			iat = 1
		}
		interArrivalTimes = append(interArrivalTimes, iat)
		totoalDuration += iat
	}

	if totoalDuration > oneMinuteInMicro {
		//* Normalise if it's longer than 1min.
		for i, iat := range interArrivalTimes {
			iat = iat / totoalDuration * oneMinuteInMicro
			if iat < 1 {
				iat = 1
			}
			interArrivalTimes[i] = iat
		}
	}

	// log.Info(stats.Sum(stats.LoadRawData(interArrivalTimes)))
	return interArrivalTimes
}

func CheckOverload(successCount, failureCount int64) bool {
	//* Amongst those returned, how many has failred?
	failureRate := float64(failureCount) / float64(successCount+failureCount)
	log.Info("Failure rate=", failureRate)
	return failureRate > OVERFLOAD_THRESHOLD
}

/**
 * This function waits for the waitgroup for the specified max timeout.
 * Returns true if waiting timed out.
 */
func wgWaitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	log.Info("Start waiting for all requests to return.")
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		log.Info("Finished waiting for all invocations.")
		return false
	case <-time.After(timeout):
		return true
	}
}

/**
 * This function has/uses side-effects, but for the sake of performance
 * keep it for now.
 */
func ShuffleAllInvocationsInplace(invocationsEachMinute *[][]int) {
	suffleOneMinute := func(invocations *[]int) {
		invRand.Shuffle(len(*invocations), func(i, j int) {
			(*invocations)[i], (*invocations)[j] = (*invocations)[j], (*invocations)[i]
		})
	}

	for minute := range *invocationsEachMinute {
		suffleOneMinute(&(*invocationsEachMinute)[minute])
	}
}

var notMedian = func() bool {
	return specRand.Int31()&0x01 == 0
}

func randIntBetween(min, max int) int {
	inverval := util.MaxOf(1, max-min)
	return specRand.Intn(inverval) + min
}

func GenerateExecutionSpecs(function tc.Function) (int, int) {

	if function.MemoryStats.Percentile100 == function.RuntimeStats.Maximum {
		//* Custom runtime specs.
		return function.RuntimeStats.Average, function.MemoryStats.Average
	}

	var runtime, memory int
	//* Generate uniform quantiles in [0, 1).
	memQtl := specRand.Float64()
	runQtl := specRand.Float64()
	//* Generate gaussian quantiles in [0, 1).
	// sigma := .25
	// mu := .5
	// memQtl := specRand.NormFloat64()*sigma + mu
	// runQtl := specRand.NormFloat64()*sigma + mu

	runStats := function.RuntimeStats
	memStats := function.MemoryStats

	switch {
	case memQtl <= 0.01:
		memory = memStats.Percentile1
	case memQtl <= 0.05:
		memory = randIntBetween(memStats.Percentile1, memStats.Percentile5)
	case memQtl <= 0.25:
		memory = randIntBetween(memStats.Percentile5, memStats.Percentile25)
	case memQtl <= 0.50:
		memory = randIntBetween(memStats.Percentile25, memStats.Percentile50)
	case memQtl <= 0.75:
		memory = randIntBetween(memStats.Percentile50, memStats.Percentile75)
	case memQtl <= 0.95:
		memory = randIntBetween(memStats.Percentile75, memStats.Percentile95)
	case memQtl <= 0.99:
		memory = randIntBetween(memStats.Percentile95, memStats.Percentile99)
	case memQtl < 1:
		memory = randIntBetween(memStats.Percentile99, memStats.Percentile100)
	}

	switch {
	case runQtl <= 0.01:
		runtime = runStats.Percentile0
	case runQtl <= 0.25:
		runtime = randIntBetween(runStats.Percentile1, runStats.Percentile25)
	case runQtl <= 0.50:
		runtime = randIntBetween(runStats.Percentile25, runStats.Percentile50)
	case runQtl <= 0.75:
		runtime = randIntBetween(runStats.Percentile50, runStats.Percentile75)
	case runQtl <= 0.95:
		runtime = randIntBetween(runStats.Percentile75, runStats.Percentile99)
	case runQtl <= 0.99:
		runtime = randIntBetween(runStats.Percentile99, runStats.Percentile100)
	case runQtl < 1:
		// 100%ile is smaller from the max. somehow.
		runtime = randIntBetween(runStats.Percentile100, runStats.Maximum)
	}

	//* Clamp specs to prevent outliers.
	runtime = util.MinOf(MAX_EXEC_TIME_MILLI, util.MaxOf(MIN_EXEC_TIME_MILLI, runtime))
	memory = util.MinOf(MAX_MEM_MIB, util.MaxOf(MIN_MEM_MIB, memory))
	return runtime, memory
}

func computeActualRps(start time.Time, counter int64) float64 {
	duration := time.Since(start).Seconds()
	return float64(counter) / duration
}

func DumpOverloadFlag() {
	// If the file doesn't exist, create it, or append to the file
	_, err := os.OpenFile("overload.flag", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
