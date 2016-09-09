package darts

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type node struct {
	code               rune /*Key_type*/
	depth, left, right int
}

type ResultPair struct {
	PrefixLen int
	Value
}

type SubWord struct {
	OffSet int
	Len    int
}
type Value struct {
	Freq     int
	SubWords []SubWord
}
type Darts struct {
	Base             []int
	Check            []int
	ValuePool        []Value
	KeyString2Object map[string]string
}

type dartsBuild struct {
	darts        Darts
	used         []bool
	size         int
	keySize      int
	key          [][]rune /*Key_type*/
	freq         []int
	nextCheckPos int
	err          int
}

// variable key should be sorted ascendingly
func Build(key [][]rune /*Key_type*/, freq []int, key2ObjectMap map[string]string) Darts {
	var d = new(dartsBuild)

	d.darts.KeyString2Object = key2ObjectMap
	d.key = key
	d.freq = freq
	d.resize(512)

	d.darts.Base[0] = 1
	d.nextCheckPos = 0

	var rootNode node
	rootNode.depth = 0
	rootNode.left = 0
	rootNode.right = len(key)

	siblings := d.fetch(rootNode)
	d.insert(siblings)

	if d.err < 0 {
		panic("Build error")
	}
	return d.darts
}

func (d *dartsBuild) resize(newSize int) {
	if newSize > len(d.darts.Base) {
		d.darts.Base = append(d.darts.Base, make([]int, (newSize-len(d.darts.Base)))...)
		d.darts.Check = append(d.darts.Check, make([]int, (newSize-len(d.darts.Check)))...)
		d.used = append(d.used, make([]bool, (newSize-len(d.used)))...)
	} else {
		d.darts.Base = d.darts.Base[:newSize]
		d.darts.Check = d.darts.Check[:newSize]
		d.used = d.used[:newSize]
	}
}

func (d *dartsBuild) fetch(parent node) []node {
	var siblings = make([]node, 0, 2)
	if d.err < 0 {
		return siblings[0:0]
	}
	var prev rune = /*Key_type*/ 0

	for i := parent.left; i < parent.right; i++ {
		if len(d.key[i]) < parent.depth {
			continue
		}

		tmp := d.key[i]

		var cur rune = /*Key_type*/ 0
		if len(d.key[i]) != parent.depth {
			cur = tmp[parent.depth] + 1
		}

		if prev > cur {
			fmt.Println(prev, cur, i, parent.depth, d.key[i])
			fmt.Println(d.key[i])
			panic("fetch error 1")
			d.err = -3
			return siblings[0:0]
		}

		if cur != prev || len(siblings) == 0 {
			var tmpNode node
			tmpNode.depth = parent.depth + 1
			tmpNode.code = cur
			tmpNode.left = i
			if len(siblings) != 0 {
				siblings[len(siblings)-1].right = i
			}

			siblings = append(siblings, tmpNode)
		}

		prev = cur
	}

	if len(siblings) != 0 {
		siblings[len(siblings)-1].right = parent.right
	}

	return siblings
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func (d *dartsBuild) insert(siblings []node) int {
	if d.err < 0 {
		panic("insert error")
		return 0
	}

	var begin int = 0
	var pos int = max(int(siblings[0].code)+1, d.nextCheckPos) - 1
	var nonZeroNum int = 0
	first := false
	if len(d.darts.Base) <= pos {
		d.resize(pos + 1)
	}

	for {
	next:
		pos++

		if len(d.darts.Base) <= pos {
			d.resize(pos + 1)
		}

		if d.darts.Check[pos] > 0 {
			nonZeroNum++
			continue
		} else if !first {
			d.nextCheckPos = pos
			first = true
		}

		begin = pos - int(siblings[0].code)
		if len(d.darts.Base) <= (begin + int(siblings[len(siblings)-1].code)) {
			d.resize(begin + int(siblings[len(siblings)-1].code) + 400)
		}

		if d.used[begin] {
			continue
		}

		for i := 1; i < len(siblings); i++ {
			if begin+int(siblings[i].code) >= len(d.darts.Base) {
				fmt.Println(len(d.darts.Base), begin+int(siblings[i].code), begin+int(siblings[len(siblings)-1].code))
			}
			if 0 != d.darts.Check[begin+int(siblings[i].code)] {
				goto next
			}
		}
		break
	}

	if float32(nonZeroNum)/float32(pos-d.nextCheckPos+1) >= 0.95 {
		d.nextCheckPos = pos
	}
	d.used[begin] = true
	d.size = max(d.size, begin+int(siblings[len(siblings)-1].code)+1)

	for i := 0; i < len(siblings); i++ {
		d.darts.Check[begin+int(siblings[i].code)] = begin
	}

	for i := 0; i < len(siblings); i++ {
		newSiblings := d.fetch(siblings[i])
		if len(newSiblings) == 0 {
			var value Value
			value.Freq = d.freq[siblings[i].left]
			d.darts.Base[begin+int(siblings[i].code)] = -len(d.darts.ValuePool) - 1
			d.darts.ValuePool = append(d.darts.ValuePool, value)
		} else {
			h := d.insert(newSiblings)
			d.darts.Base[begin+int(siblings[i].code)] = h
		}
	}

	return begin
}
func (d Darts) UpdateThesaurus(keys [][]rune /*Key_type*/) {
f0:
	for _, key := range keys {
		wordLen := len(key)
		if wordLen < 2 {
			continue
		}

		var subWords []SubWord
		for i := 0; i < wordLen-1; i++ {
			results := d.CommonPrefixSearch(key[i:], 0)
			for _, result := range results {
				if result.PrefixLen > 1 && result.PrefixLen < wordLen {
					subWords = append(subWords, SubWord{i, result.PrefixLen})
				}
			}
		}

		b := d.Base[0]
		var p int

		for i := 0; i < len(key); i++ {
			p = b + int(key[i]) + 1
			if b == d.Check[p] {
				b = d.Base[p]
			} else {
				continue f0
			}
		}

		p = b
		n := d.Base[p]
		if b == d.Check[p] && n < 0 {
			d.ValuePool[-n-1].SubWords = subWords
		}
	}
}
func (d Darts) ExactMatchSearch(key []rune /*Key_type*/, nodePos int) bool {
	b := d.Base[nodePos]
	var p int

	for i := 0; i < len(key); i++ {
		p = b + int(key[i]) + 1
		if b == d.Check[p] {
			b = d.Base[p]
		} else {
			return false
		}
	}

	p = b
	n := d.Base[p]
	if b == d.Check[p] && n < 0 {
		return true
	}

	return false
}
func (d Darts) CommonPrefixSearch(key []rune /*Key_type*/, nodePos int) (results []ResultPair) {
	b := d.Base[nodePos]
	var p int

	for i := 0; i < len(key); i++ {
		p = b
		n := d.Base[p]
		if b == d.Check[p] && n < 0 {
			results = append(results, ResultPair{i, d.ValuePool[-n-1]})
		}

		p = b + int(key[i]) + 1
		// p could be bigger than the index
		if p >= len(d.Check) {
			return results
		}

		if b == d.Check[p] {
			b = d.Base[p]
		} else {
			return results
		}
	}

	p = b
	n := d.Base[p]
	if b == d.Check[p] && n < 0 {
		results = append(results, ResultPair{len(key), d.ValuePool[-n-1]})
	}
	return results
}
func (d *Darts) ContainMatch(key []rune, nodePos int) (bool, error) {
	var flag bool
	var rst []ResultPair
	word := key
	if len(word) == 0 {
		flag = false
		goto END_MATCH
	}
CONTINUE_MATCH:
	rst = d.CommonPrefixSearch(word, 0)
	if rst == nil {
		if len(word) < 2 {
			flag = false
			goto END_MATCH
		}
		word = word[1:]
		goto CONTINUE_MATCH
	}
	return true, nil
END_MATCH:
	return flag, nil
}
func (d *Darts) MultiSearch(key []rune, nodePos int) ([]string, error) {
	var rst []ResultPair
	var keyword string
	var index int
	var object []string
	var oid string
	checked := make(map[string]bool)
	oidChecked := make(map[string]bool)
	word := key
	if len(word) == 0 {
		goto END
	}
MATCH:
	if 0 == len(word) {
		goto END
	}
	rst = d.CommonPrefixSearch(word, 0)
	if 0 == len(rst) {
		if len(word) < 2 {
			goto END
		}
		word = word[1:]
		goto MATCH
	}
	index = rst[len(rst)-1].PrefixLen
	keyword = string(word[:index])
	// fmt.Println(d.KeyString2Object[keyword])
	oid = d.KeyString2Object[keyword]
	if false == checked[keyword] && false == oidChecked[oid] {
		object = append(object, oid)
		checked[keyword] = true
		oidChecked[oid] = true
	}
	word = word[index:]
	goto MATCH
END:
	return object, nil
}
func Load(filename string) (Darts, error) {
	var dict Darts
	file, err := os.Open(filename)
	if err != nil {
		return Darts{}, err
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	dec.Decode(&dict)
	return dict, nil
}

type dartsKey struct {
	key   []rune /*Key_type*/
	value int
	Oid   string
}
type dartsKeySlice []dartsKey

func (r dartsKeySlice) Len() int {
	return len(r)
}
func (r dartsKeySlice) Less(i, j int) bool {
	var l int
	if len(r[i].key) < len(r[j].key) {
		l = len(r[i].key)
	} else {
		l = len(r[j].key)
	}

	for m := 0; m < l; m++ {
		if r[i].key[m] < r[j].key[m] {
			return true
		} else if r[i].key[m] == r[j].key[m] {
			continue
		} else {
			return false
		}
	}
	if len(r[i].key) < len(r[j].key) {
		return true
	}
	return false
}
func (r dartsKeySlice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func Import(inFile, outFile string, useDAWG bool) (Darts, error) {
	unifile, erri := os.Open(inFile)
	if erri != nil {
		return Darts{}, erri
	}
	defer unifile.Close()
	ofile, erro := os.Create(outFile)
	if erro != nil {
		return Darts{}, erro
	}
	defer ofile.Close()

	dartsKeys := make(dartsKeySlice, 0, 130000)
	uniLineReader := bufio.NewReaderSize(unifile, 400)
	line, _, bufErr := uniLineReader.ReadLine()
	for nil == bufErr {
		rst := strings.Split(string(line), "\t")
		//todo 错误处理
		key := []rune(rst[0])
		value, _ := strconv.Atoi(rst[1])
		oid := strings.TrimSpace(rst[2])
		if 0 != len(key) {
			dartsKeys = append(dartsKeys, dartsKey{key, value, oid})
		}
		line, _, bufErr = uniLineReader.ReadLine()
	}
	sort.Sort(dartsKeys)

	keys := make([][]rune, len(dartsKeys))
	values := make([]int, len(dartsKeys))
	keyString2Object := make(map[string]string)

	for i := 0; i < len(dartsKeys); i++ {
		keys[i] = dartsKeys[i].key
		values[i] = dartsKeys[i].value
		keyString2Object[string(dartsKeys[i].key)] = dartsKeys[i].Oid
	}
	fmt.Printf("input dict length: %v\n", len(keys))
	round := len(keys)
	var d Darts
	if useDAWG {
		d = BuildFromDAWG(keys[:round], values[:round])
	} else {
		d = Build(keys[:round], values[:round], keyString2Object)
	}
	d.UpdateThesaurus(keys[:round])
	fmt.Printf("build out length %v\n", len(d.Base))
	t := time.Now()
	for i := 0; i < round; i++ {
		if true != d.ExactMatchSearch(keys[i], 0) {
			err := fmt.Errorf("missing key %s, %v, %d, %v, %v", string(keys[i]), keys[i], i, keys[i-1], keys[i+1])
			return d, err
		}
	}
	fmt.Println(time.Since(t))
	enc := gob.NewEncoder(ofile)
	enc.Encode(d)

	return d, nil
}

func Generate(list []map[string]string, useDAWG bool) (Darts, error) {
	if 0 == len(list) {
		return Darts{}, fmt.Errorf("empty list input")
	}
	dartsKeys := make(dartsKeySlice, 0, 130000)
	for _,item := range list {
		//rst := strings.Split(string(line), "\t")
		key := []rune(item["key"])
		value, _ := strconv.Atoi(item["value"])
		oid := strings.TrimSpace(item["oid"])
		if 0 != len(key) {
			dartsKeys = append(dartsKeys, dartsKey{key, value, oid})
		}
	}
	sort.Sort(dartsKeys)

	keys := make([][]rune, len(dartsKeys))
	values := make([]int, len(dartsKeys))
	keyString2Object := make(map[string]string)

	for i := 0; i < len(dartsKeys); i++ {
		keys[i] = dartsKeys[i].key
		values[i] = dartsKeys[i].value
		keyString2Object[string(dartsKeys[i].key)] = dartsKeys[i].Oid
	}
	fmt.Printf("input dict length: %v\n", len(keys))
	round := len(keys)
	var d Darts
	if useDAWG {
		d = BuildFromDAWG(keys[:round], values[:round])
	} else {
		d = Build(keys[:round], values[:round], keyString2Object)
	}
	d.UpdateThesaurus(keys[:round])
	fmt.Printf("build out length %v\n", len(d.Base))
	t := time.Now()
	for i := 0; i < round; i++ {
		if true != d.ExactMatchSearch(keys[i], 0) {
			err := fmt.Errorf("missing key %s, %v, %d, %v, %v", string(keys[i]), keys[i], i, keys[i-1], keys[i+1])
			return d, err
		}
	}
	fmt.Println(time.Since(t))
	//enc := gob.NewEncoder(ofile)
	//enc.Encode(d)

	return d, nil
}
