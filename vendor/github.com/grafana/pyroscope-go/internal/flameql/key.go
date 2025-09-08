package flameql

import (
	"strings"

	"github.com/grafana/pyroscope-go/internal/sortedmap"
)

type Key struct {
	labels map[string]string
}

type ParserState int

const (
	nameParserState ParserState = iota
	tagKeyParserState
	tagValueParserState
	doneParserState
)

func NewKey(labels map[string]string) *Key { return &Key{labels: labels} }

func ParseKey(name string) (*Key, error) {
	k := &Key{labels: make(map[string]string)}
	p := parser{parserState: nameParserState}
	var err error
	for _, r := range name + "{" {
		switch p.parserState {
		case nameParserState:
			err = p.nameParserCase(r, k)
		case tagKeyParserState:
			p.tagKeyParserCase(r)
		case tagValueParserState:
			err = p.tagValueParserCase(r, k)
		case doneParserState:
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}

	return k, nil
}

type parser struct {
	parserState ParserState
	key         string
	value       string
}

// ParseKey's nameParserState switch case
func (p *parser) nameParserCase(r int32, k *Key) error {
	switch r {
	case '{':
		p.parserState = tagKeyParserState
		appName := strings.TrimSpace(p.value)
		if err := ValidateAppName(appName); err != nil {
			return err
		}
		k.labels["__name__"] = appName
	default:
		p.value += string(r)
	}

	return nil
}

// ParseKey's tagKeyParserState switch case
func (p *parser) tagKeyParserCase(r int32) {
	switch r {
	case '}':
		p.parserState = doneParserState
	case '=':
		p.parserState = tagValueParserState
		p.value = ""
	default:
		p.key += string(r)
	}
}

// ParseKey's tagValueParserState switch case
func (p *parser) tagValueParserCase(r int32, k *Key) error {
	switch r {
	case ',', '}':
		p.parserState = tagKeyParserState
		key := strings.TrimSpace(p.key)
		if !IsTagKeyReserved(key) {
			if err := ValidateTagKey(key); err != nil {
				return err
			}
		}
		k.labels[key] = strings.TrimSpace(p.value)
		p.key = ""
	default:
		p.value += string(r)
	}

	return nil
}

func (k *Key) Normalized() string {
	var sb strings.Builder

	sortedMap := sortedmap.New()
	for k, v := range k.labels {
		if k == "__name__" {
			sb.WriteString(v)
		} else {
			sortedMap.Put(k, v)
		}
	}

	sb.WriteString("{")
	for i, k := range sortedMap.Keys() {
		v := sortedMap.Get(k)
		if i != 0 {
			sb.WriteString(",")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
	}
	sb.WriteString("}")

	return sb.String()
}

func (k *Key) AppName() string {
	return k.labels["__name__"]
}

func (k *Key) Labels() map[string]string {
	return k.labels
}

func (k *Key) Add(key, value string) {
	if value == "" {
		delete(k.labels, key)
	} else {
		k.labels[key] = value
	}
}

func (k *Key) Clone() *Key {
	newMap := make(map[string]string)
	for k, v := range k.labels {
		newMap[k] = v
	}

	return &Key{labels: newMap}
}
