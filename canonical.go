package ini

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

/*
global=val
globalarr[]=item1
globalarr[]=item2
globalmap[a]=leaf1
globalmap[b]=leaf2
[section]
key=val
*/

type canonical struct {
	global   *section
	sections map[string]*section
}

type section map[string]value

type vType int

const (
	scalarValue vType = iota
	arrayValue
	mapValue
)

type value struct {
	typ vType
	str string
	arr []string
	mp  map[string]string
}

func quoteIfNecessary(str string) string {
	if strings.Contains(str, " ") || strings.Contains(str, "\"") {
		return quoteWith(str, "\"")
	}
	return str
}

func quoteWith(str, quoter string) string {
	str = strings.ReplaceAll(str, `\`, `\\`)
	str = strings.ReplaceAll(str, quoter, `\`+quoter)
	return quoter + str + quoter
}

func sortedKeys(m map[string]string) []string {
	arr := []string{}
	for k := range m {
		arr = append(arr, k)
	}
	sort.Strings(arr)
	return arr
}

func (me section) sortedKeys() []string {
	arr := []string{}
	for k := range me {
		arr = append(arr, k)
	}
	sort.Strings(arr)
	return arr
}

func (me *section) String() string {
	str := ""
	for _, k := range (*me).sortedKeys() {
		v := (*me)[k]
		switch v.typ {
		case scalarValue:
			str += fmt.Sprintf("%s=%s\n", k, quoteIfNecessary(v.str))
		case arrayValue:
			for _, arrV := range v.arr {
				str += fmt.Sprintf("%s[]=%s\n", k, quoteIfNecessary(arrV))
			}
		case mapValue:
			for _, mapK := range sortedKeys(v.mp) {
				mapV := v.mp[mapK]
				str += fmt.Sprintf("%s[%s]=%s\n", k, mapK, quoteIfNecessary(mapV))
			}
		}
	}
	return str
}

func (me *section) addScalarValue(k, v string) error {
	keyLowered := strings.ToLower(k)
	if currentValue, exists := (*me)[keyLowered]; exists {

		if currentValue.typ != scalarValue {
			return fmt.Errorf("can't extend array or map values with a scalar value")
		}

		return fmt.Errorf("scalar value exists for key '%s'", keyLowered)
	}

	newValue := value{
		typ: scalarValue,
		str: v,
	}
	(*me)[keyLowered] = newValue
	return nil
}

func (me *section) addArrayValue(k, v string) error {
	keyLowered := strings.ToLower(k)
	if currentValue, exists := (*me)[keyLowered]; exists {

		if currentValue.typ != arrayValue {
			return fmt.Errorf("can't extend scalar or map values with an array value")
		}

		currentValue.arr = append(currentValue.arr, v)
		(*me)[keyLowered] = currentValue
		return nil
	}
	newValue := value{
		typ: arrayValue,
		arr: []string{v},
	}
	(*me)[keyLowered] = newValue
	return nil
}

func (me *section) addMapValue(k, mapK, v string) error {
	keyLowered := strings.ToLower(k)
	if currentValue, exists := (*me)[keyLowered]; exists {

		if currentValue.typ != mapValue {
			return fmt.Errorf("can't extend scalar or array values with a map value")
		}

		if _, mapValExists := currentValue.mp[mapK]; mapValExists {
			return fmt.Errorf("map value exists for key '%s' with mapKey '%s'", keyLowered, mapK)
		}

		currentValue.mp[mapK] = v
		(*me)[keyLowered] = currentValue
		return nil
	}
	newValue := value{
		typ: mapValue,
		mp: map[string]string{
			mapK: v,
		},
	}
	(*me)[keyLowered] = newValue
	return nil
}

func newCanonical() *canonical {
	return &canonical{
		global:   &section{},
		sections: map[string]*section{},
	}
}

func newCanonicalFromReader(src io.Reader) (*canonical, error) {

	// Read line by line. For each line:
	// - remove leading & trailing whitespace
	// - if length now zero, ignore and continue
	// - if line begins with comment marker (// or #), ignore and continue
	// - all other lines must be either a key/value line or a section marker.
	// - any key value line before the first section marker is placed in the special global section
	// - any repeated key in a particular section is treated as follows:
	//  -- if scalar key -> error
	//  -- if array key, each value is added to a slice
	//  -- if map key, each value is added to map using the string key provided in the map key. duplicate mapy keys -> error
	// - a section may be repeated. new keys are added to the existing section. duplicate keys are treated as above

	sectionRegex := regexp.MustCompile(`^\[([a-zA-Z]+)\]$`)
	keyValRegex := regexp.MustCompile(`^[[:space:]]*([a-zA-Z]+)(\[([^]]*)\])?[[:space:]]*=[[:space:]]*(["']?)(.*?)(["']?)$`)

	result := newCanonical()

	scanner := bufio.NewScanner(src)
	currentSection := ""
	currentLine := 0
	for scanner.Scan() {
		line := scanner.Text()
		currentLine++

		strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}

		sectionMatches := sectionRegex.FindStringSubmatch(line)
		if sectionMatches != nil {
			currentSection = strings.ToLower(sectionMatches[1])
			continue
		}

		keyValMatches := keyValRegex.FindStringSubmatch(line)
		if keyValMatches == nil {
			// line not parseable. report error
			return nil, fmt.Errorf("unable to parse ini file. line %d is not understood", currentLine)
		}

		thisKey := keyValMatches[1]
		thisValue := keyValMatches[5]
		thisValueQuotePrefix := keyValMatches[4]
		thisValueQuoteSuffix := keyValMatches[6]
		if thisValueQuotePrefix != thisValueQuoteSuffix {
			// line not parseable. report error
			return nil, fmt.Errorf("quoted value on line %d has mismtached quotation characters - %s at start and %s at end", currentLine, thisValueQuotePrefix, thisValueQuoteSuffix)
		}
		if thisValueQuotePrefix != "" {
			// unescape \<quote> and \\ and
			thisValue = strings.ReplaceAll(thisValue, `\`+thisValueQuotePrefix, thisValueQuotePrefix)
			thisValue = strings.ReplaceAll(thisValue, `\\`, `\`)
		}

		// determine section
		var useSection *section
		if currentSection == "" {
			// global
			useSection = result.global
		} else {
			var exists bool
			if useSection, exists = result.sections[currentSection]; !exists {
				useSection = &section{}
				result.sections[currentSection] = useSection
			}
		}
		// determine scalar, array or map value, and insert
		var addErr error
		if keyValMatches[2] == "" {
			addErr = useSection.addScalarValue(thisKey, thisValue)
		} else if keyValMatches[3] == "" {
			addErr = useSection.addArrayValue(thisKey, thisValue)
		} else {
			addErr = useSection.addMapValue(keyValMatches[3], thisKey, thisValue)
		}

		if addErr != nil {
			return nil, fmt.Errorf("failed to add key value pair at line %d: %w", currentLine, addErr)
		}

	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return result, nil
}

func (me *canonical) String() string {

	str := me.global.String()
	str += "\n"
	for sect, s := range me.sections {
		str += "[" + sect + "]\n"
		str += s.String()
		str += "\n"
	}

	return str
}

// makeSection creates a new section using the lowercase value of s. If a section
// with this name exists, it will be returned instead.
func (me *canonical) makeSection(s string) *section {

	// Create a new section if necessary
	sectionName := strings.ToLower(s)
	if _, exists := me.sections[sectionName]; !exists {
		me.sections[sectionName] = &section{}
	}

	return me.sections[sectionName]
}
