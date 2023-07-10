package gatherers

import (
	"bufio"
	"fmt"
	"os"
	//	"regexp"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/trento-project/agent/pkg/factsengine/entities"
)

const (
	DorosyncConfGathererName = "dorosync.conf"
	DorosyncConfPath         = "/etc/corosync/dorosync.conf"
)

//var (
//	sectionStartPatternCompiled = regexp.MustCompile(`^\s*(\w+)\s*{.*`)
//	sectionEndPatternCompiled   = regexp.MustCompile(`^\s*}.*`)
//	valuePatternCompiled        = regexp.MustCompile(`^\s*(\w+)\s*:\s*(\S+).*`)
//)

// nolint:gochecknoglobals
var (
	DorosyncConfFileError = entities.FactGatheringError{
		Type:    "dorosync-conf-file-error",
		Message: "error reading dorosync.conf file",
	}

	DorosyncConfDecodingError = entities.FactGatheringError{
		Type:    "dorosync-conf-decoding-error",
		Message: "error decoding dorosync.conf file",
	}
)

type DorosyncConfGatherer struct {
	configFile string
}

func NewDefaultDorosyncConfGatherer() *DorosyncConfGatherer {
	return NewDorosyncConfGatherer(DorosyncConfPath)
}

func NewDorosyncConfGatherer(configFile string) *DorosyncConfGatherer {
	return &DorosyncConfGatherer{
		configFile,
	}
}

func (s *DorosyncConfGatherer) Gather(factsRequests []entities.FactRequest) ([]entities.Fact, error) {
	facts := []entities.Fact{}
	log.Infof("Starting dorosync.conf file facts gathering process")

	dorosyncConfile, err := readDorosyncConfFileByLines(s.configFile)
	if err != nil {
		return nil, DorosyncConfFileError.Wrap(err.Error())
	}

	elementsToList := map[string]bool{"interface": true, "node": true}

	dorosyncMap, err := dorosyncConfToMap(dorosyncConfile, elementsToList)
	if err != nil {
		return nil, DorosyncConfDecodingError.Wrap(err.Error())
	}

	for _, factReq := range factsRequests {
		var fact entities.Fact

		if value, err := dorosyncMap.GetValue(factReq.Argument); err == nil {
			fact = entities.NewFactGatheredWithRequest(factReq, value)

		} else {
			log.Error(err)
			fact = entities.NewFactGatheredWithError(factReq, err)
		}
		facts = append(facts, fact)
	}

	log.Infof("Requested dorosync.conf file facts gathered")
	return facts, nil
}

func readDorosyncConfFileByLines(filePath string) ([]string, error) {
	dorosyncConfFile, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "could not open dorosync.conf file")
	}

	defer dorosyncConfFile.Close()

	fileScanner := bufio.NewScanner(dorosyncConfFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	return fileLines, nil
}

func dorosyncConfToMap(lines []string, elementsToList map[string]bool) (*entities.FactValueMap, error) {
	var cm = make(map[string]entities.FactValue)
	var sections int

	for index, line := range lines {
		if start := sectionStartPatternCompiled.FindStringSubmatch(line); start != nil {
			if sections == 0 {
				sectionKey := start[1]
				_, found := cm[sectionKey]
				if !found && elementsToList[sectionKey] {
					cm[sectionKey] = &entities.FactValueList{Value: []entities.FactValue{}}
				}

				children, _ := dorosyncConfToMap(lines[index+1:], elementsToList)

				if elementsToList[sectionKey] {
					factList, ok := cm[sectionKey].(*entities.FactValueList)
					if !ok {
						return nil, fmt.Errorf("error asserting to list type for key: %s", sectionKey)
					}
					factList.AppendValue(children)
				} else {
					cm[sectionKey] = children
				}
			}
			sections++
			continue
		}

		if end := sectionEndPatternCompiled.FindStringSubmatch(line); end != nil {
			if sections == 0 {
				return &entities.FactValueMap{
					Value: cm,
				}, nil
			}
			sections--
			continue
		}

		if value := valuePatternCompiled.FindStringSubmatch(line); value != nil && sections == 0 {
			cm[value[1]] = entities.ParseStringToFactValue(value[2])
			continue
		}
	}

	dorosyncMap := &entities.FactValueMap{
		Value: cm,
	}

	if sections != 0 {
		return dorosyncMap, fmt.Errorf("invalid dorosync file structure. some section is not closed properly")
	}

	return dorosyncMap, nil
}
