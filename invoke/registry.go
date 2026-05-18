package invoke

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
)

var (
	methodList     []string
	methodListOnce sync.Once
)

func MethodList() []string {
	methodListOnce.Do(func() {
		methodList = make([]string, 0, len(tg.NamesMap))
		for name := range tg.NamesMap {
			if _, ok := tg.FunctionsMap[tg.NamesMap[name]]; ok {
				methodList = append(methodList, name)
			}
		}
		sort.Strings(methodList)
	})
	return methodList
}

func FilterMethods(prefix string) []string {
	all := MethodList()
	if prefix == "" {
		return all
	}
	var result []string
	for _, m := range all {
		if strings.HasPrefix(m, prefix) {
			result = append(result, m)
		}
	}
	return result
}

func MethodExists(name string) bool {
	id, ok := tg.NamesMap[name]
	if !ok {
		return false
	}
	_, ok = tg.FunctionsMap[id]
	return ok
}

func GetMethodID(name string) (uint32, error) {
	id, ok := tg.NamesMap[name]
	if !ok {
		return 0, fmt.Errorf("unknown TL method: %s", name)
	}
	return id, nil
}
