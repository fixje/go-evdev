package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type kv map[string]string

var defs map[string]kv

func stringInSlice(s string, slice []string) bool {
	for _, x := range slice {
		if x == s {
			return true
		}
	}

	return false
}

func readfile(fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}

	typeBlacklist := []string{
		"EVIOCGKEYCODE",
		"EVIOCSKEYCODE",
	}

	nameBlacklist := []string{
		"KEYMAP_BY_INDEX",
		"KEYMAP_BY_INDEX",
		"PROP_CNT",
		"VERSION",
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := scanner.Text()

		re := regexp.MustCompile(`^#define[ \t]([A-Z]+)_([A-Z0-9_]+)[ \t]+([0-9a-fA-Zx()_+ ]+)`)
		matches := re.FindAllStringSubmatch(s, -1)

		if len(matches) != 1 {
			continue
		}

		submatches := matches[0]
		varType := submatches[1]
		varName := submatches[2]
		varValue := submatches[3]

		if stringInSlice(varType, typeBlacklist) {
			continue
		}

		if stringInSlice(varName, nameBlacklist) {
			continue
		}

		if varType == "INPUT" && strings.HasPrefix(varName, "PROP_") {
			varType = "PROP"
			varName = strings.TrimLeft(varName, "PROP_")
		}

		content, ok := defs[varType]
		if !ok {
			content = make(kv)
		}

		content[varName] = varValue
		defs[varType] = content
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func sortedIntKeys(m map[int]string) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	return keys
}

func main() {
	defs = make(map[string]kv)

	for _, f := range os.Args[1:] {
		err := readfile(f)
		if err != nil {
			fmt.Printf("Cannot read %s: %v", f, err)
			return
		}
	}

	fmt.Printf("package evdev\n\n")

	fmt.Printf("\n")
	fmt.Printf("//\n")
	fmt.Printf("// THIS FILE IS AUTO-GENERATED. DO NOT MODIFY MANUALLY\n")
	fmt.Printf("//\n")
	fmt.Printf("\n")

	for dType, dContent := range defs {
		fmt.Printf("// %s\n", dType)
		fmt.Printf("const (\n")

		for _, k := range sortedStringKeys(dContent) {
			fmt.Printf("\t%s_%s = %s\n", dType, k, dContent[k])
		}

		fmt.Printf(")\n")

		m := make(map[int]string)
		for k, v := range dContent {
			a := strings.Split(v, " ")
			n, err := strconv.ParseInt(a[0], 0, 32)
			if k != "MAX" && err == nil {
				// FIXME: handle duplicates
				m[int(n)] = k
			}
		}

		var mType string

		switch dType {
		case "EV":
			mType = "EvType"
		case "PROP":
			mType = "EvProp"
		default:
			mType = "EvCode"
		}

		if dType == "EV" {
			mType = "EvType"
		}

		fmt.Printf("var %sName = map[%s]string {\n", strings.Title(dType), mType)
		for _, k := range sortedIntKeys(m) {
			fmt.Printf("\t%d: \"%s_%s\",\n", k, dType, m[k])
		}
		fmt.Printf("}\n")
	}
}
