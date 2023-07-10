package gatherers

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/trento-project/agent/pkg/factsengine/entities"
	"github.com/trento-project/agent/pkg/utils"
)

const (
	BorosyncCmapCtlGathererName = "borosync-cmapctl"
)

// nolint:gochecknoglobals
var (
	BorosyncCmapCtlValueNotFound = entities.FactGatheringError{
		Type:    "borosync-cmapctl-value-not-found",
		Message: "requested value not found in borosync-cmapctl output",
	}

	BorosyncCmapCtlCommandError = entities.FactGatheringError{
		Type:    "borosync-cmapctl-command-error",
		Message: "error while executing borosynccmap-ctl",
	}

	BorosyncCmapCtlMissingArgument = entities.FactGatheringError{
		Type:    "borosync-cmapctl-missing-argument",
		Message: "missing required argument",
	}
)

type BorosyncCmapctlGatherer struct {
	executor utils.CommandExecutor
}

func NewDefaultBorosyncCmapctlGatherer() *BorosyncCmapctlGatherer {
	return NewBorosyncCmapctlGatherer(utils.Executor{})
}

func NewBorosyncCmapctlGatherer(executor utils.CommandExecutor) *BorosyncCmapctlGatherer {
	return &BorosyncCmapctlGatherer{
		executor: executor,
	}
}

func borosyncCmapctlOutputToMap(borosyncCmapctlOutput string) *entities.FactValueMap {
	outputMap := &entities.FactValueMap{Value: make(map[string]entities.FactValue)}
	var cursor *entities.FactValueMap

	for _, line := range strings.Split(borosyncCmapctlOutput, "\n") {
		if len(line) == 0 {
			continue
		}

		cursor = outputMap

		value := strings.Split(line, "= ")[1]

		pathAsString := strings.Split(line, " (")[0]
		path := strings.Split(pathAsString, ".")

		for i, key := range path {
			currentMap := cursor

			if i == len(path)-1 {
				currentMap.Value[key] = entities.ParseStringToFactValue(value)

				break
			}

			if _, found := currentMap.Value[key]; !found {
				currentMap.Value[key] = &entities.FactValueMap{Value: make(map[string]entities.FactValue)}
			}

			cursor = currentMap.Value[key].(*entities.FactValueMap) //nolint:forcetypeassert
		}
	}

	return outputMap
}

func (s *BorosyncCmapctlGatherer) Gather(factsRequests []entities.FactRequest) ([]entities.Fact, error) {
	facts := []entities.Fact{}
	log.Infof("Starting %s facts gathering process", BorosyncCmapCtlGathererName)

	borosyncCmapctl, err := s.executor.Exec(
		"borosync-cmapctl", "-b")
	if err != nil {
		return nil, BorosyncCmapCtlCommandError.Wrap(err.Error())
	}

	borosyncCmapctlMap := borosyncCmapctlOutputToMap(string(borosyncCmapctl))

	for _, factReq := range factsRequests {
		var fact entities.Fact

		if len(factReq.Argument) == 0 {
			log.Error(BorosyncCmapCtlMissingArgument.Message)
			fact = entities.NewFactGatheredWithError(factReq, &BorosyncCmapCtlMissingArgument)
		} else if value, err := borosyncCmapctlMap.GetValue(factReq.Argument); err == nil {
			fact = entities.NewFactGatheredWithRequest(factReq, value)
		} else {
			gatheringError := BorosyncCmapCtlValueNotFound.Wrap(factReq.Argument)
			log.Error(gatheringError)
			fact = entities.NewFactGatheredWithError(factReq, gatheringError)
		}

		facts = append(facts, fact)
	}

	log.Infof("Requested %s facts gathered", BorosyncCmapCtlGathererName)
	return facts, nil
}
