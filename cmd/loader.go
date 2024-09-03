package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/vhive-serverless/loader/pkg/common"
	"github.com/vhive-serverless/loader/pkg/config"
	"github.com/vhive-serverless/loader/pkg/driver"
	"github.com/vhive-serverless/loader/pkg/generator"
	"github.com/vhive-serverless/loader/pkg/trace"
	"golang.org/x/exp/slices"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	tracer "github.com/vhive-serverless/vSwarm/utils/tracing/go"
)

const (
	zipkinAddr = "http://localhost:9411/api/v2/spans"
)

var (
	configPath    = flag.String("config", "config.json", "Path to loader configuration file")
	verbosity     = flag.String("verbosity", "info", "Logging verbosity - choose from [info, debug, trace]")
	iatGeneration = flag.Bool("iatGeneration", false, "Generate iats only or run invocations as well")
	iatFromFile   = flag.Bool("generated", false, "True if iats were already generated")
)

func init() {
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.StampMilli,
		FullTimestamp:   true,
	})
	log.SetOutput(os.Stdout)

	switch *verbosity {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}
func main() {
	cfg := config.ReadConfigurationFile(*configPath)
	if cfg.EnableZipkinTracing {
		// TODO: how not to exclude Zipkin spans here? - file a feature request
		log.Warnf("Zipkin tracing has been enabled. This will exclude Istio spans from the Zipkin traces.")
		shutdown, err := tracer.InitBasicTracer(zipkinAddr, "loader")
		if err != nil {
			log.Print(err)
		}
		defer shutdown()
	}
	if cfg.ExperimentDuration < 1 {
		log.Fatal("Runtime duration should be longer, at least a minute.")
	}

	if *iatGeneration {
		durationToParse := determineDurationToParse(cfg.ExperimentDuration, cfg.WarmupDuration)
		iatDistribution, shiftIAT := parseIATDistribution(&cfg)
		traceParser := trace.NewAzureParser(cfg.TracePath, durationToParse)
		functions := traceParser.Parse(cfg.Platform)

		justGenerateIAT(cfg.Seed, iatDistribution, shiftIAT, parseTraceGranularity(&cfg), functions)
	}

	supportedPlatforms := []string{
		"Knative",
		"Knative-RPS",
		"OpenWhisk",
		"OpenWhisk-RPS",
		"AWSLambda",
		"AWSLambda-RPS",
		"Dirigent",
		"Dirigent-RPS",
		"Dirigent-Dandelion-RPS",
		"Dirigent-Dandelion",
	}

	if !slices.Contains(supportedPlatforms, cfg.Platform) {
		log.Fatal("Unsupported platform!")
	}

	if !strings.HasSuffix(cfg.Platform, "-RPS") {
		runTraceMode(&cfg, *iatFromFile)
	} else {
		runRPSMode(&cfg)
	}
}

func determineDurationToParse(runtimeDuration int, warmupDuration int) int {
	result := 0

	if warmupDuration > 0 {
		result += 1              // profiling
		result += warmupDuration // warmup
	}

	result += runtimeDuration // actual experiment

	return result
}

func parseIATDistribution(cfg *config.LoaderConfiguration) (common.IatDistribution, bool) {
	switch cfg.IATDistribution {
	case "exponential":
		return common.Exponential, false
	case "exponential_shift":
		return common.Exponential, true
	case "uniform":
		return common.Uniform, false
	case "uniform_shift":
		return common.Uniform, true
	case "equidistant":
		return common.Equidistant, false
	default:
		log.Fatal("Unsupported IAT distribution.")
	}

	return common.Exponential, false
}

func parseYAMLSpecification(cfg *config.LoaderConfiguration) string {
	switch cfg.YAMLSelector {
	case "container":
		return "workloads/container/trace_func_go.yaml"
	case "firecracker":
		return "workloads/firecracker/trace_func_go.yaml"
	default:
		if cfg.Platform != "Dirigent" && cfg.Platform != "Dirigent-RPS" && cfg.Platform != "Dirigent-Dandelion-RPS" && cfg.Platform != "Dirigent-Dandelion" {
			log.Fatal("Invalid 'YAMLSelector' parameter.")
		}
	}

	return ""
}

func parseTraceGranularity(cfg *config.LoaderConfiguration) common.TraceGranularity {
	switch cfg.Granularity {
	case "minute":
		return common.MinuteGranularity
	case "second":
		return common.SecondGranularity
	default:
		log.Fatal("Invalid trace granularity parameter.")
	}

	return common.MinuteGranularity
}

func runTraceMode(cfg *config.LoaderConfiguration, readIATFromFile bool) {
	durationToParse := determineDurationToParse(cfg.ExperimentDuration, cfg.WarmupDuration)
	yamlPath := parseYAMLSpecification(cfg)

	traceParser := trace.NewAzureParser(cfg.TracePath, durationToParse)
	functions := traceParser.Parse(cfg.Platform)

	log.Infof("Traces contain the following %d functions:\n", len(functions))
	for _, function := range functions {
		fmt.Printf("\t%s\n", function.Name)
	}

	iatType, shiftIAT := parseIATDistribution(cfg)

	experimentDriver := driver.NewDriver(&config.Configuration{
		LoaderConfiguration: cfg,
		IATDistribution:     iatType,
		ShiftIAT:            shiftIAT,
		TraceGranularity:    parseTraceGranularity(cfg),
		TraceDuration:       durationToParse,

		YAMLPath: yamlPath,
		TestMode: false,

		Functions: functions,
	})

	log.Infof("Using %s as a service YAML specification file.\n", experimentDriver.Configuration.YAMLPath)

	experimentDriver.RunExperiment(false, readIATFromFile)
}

func justGenerateIAT(seed int64, iatDistribution common.IatDistribution, shiftIAT bool, traceGranularity common.TraceGranularity, functions []*common.Function) {
	specificationGenerator := generator.NewSpecificationGenerator(seed)

	for i, function := range functions {
		spec := specificationGenerator.GenerateInvocationData(
			function,
			iatDistribution,
			shiftIAT,
			traceGranularity,
		)
		functions[i].Specification = spec

		file, _ := json.MarshalIndent(spec, "", " ")
		err := os.WriteFile("iat"+strconv.Itoa(i)+".json", file, 0644)
		if err != nil {
			log.Fatalf("Writing the loader config file failed: %s", err)
		}
	}
}

func runRPSMode(cfg *config.LoaderConfiguration) {
	panic("Not yet implemented")
}
