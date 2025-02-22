package glang

import (
	"bytes"
	"fmt"
	"strings"
)

type Dictionary struct {
	Name string

	OverwriteAllowed bool

	tabSpaces string

	entries map[string]string
	keyBuf  []byte

	replacer *strings.Replacer
}

type DictionaryConfig struct {
	Name string

	TabSpaces int

	Replacements map[string]string
}

func NewDictionary(config DictionaryConfig) *Dictionary {
	d := &Dictionary{
		Name:      config.Name,
		entries:   make(map[string]string, 64),
		keyBuf:    make([]byte, 256),
		tabSpaces: strings.Repeat(" ", config.TabSpaces),
	}
	if len(config.Replacements) != 0 {
		var pairs []string
		for key, val := range config.Replacements {
			pairs = append(pairs, key, val)
		}
		d.replacer = strings.NewReplacer(pairs...)
	}
	return d
}

func ParseDictionary(config DictionaryConfig, data []byte) (*Dictionary, error) {
	dict := NewDictionary(config)
	err := dict.Load("", data)
	return dict, err
}

func (d *Dictionary) Load(prefix string, data []byte) error {
	offset := 0
	sectionBodyBegin := 0
	sectionBodyEnd := 0
	sectionKey := ""
	nextSectionBodyBegin := 0
	nextSectionKey := ""
	for {
		lineEnd := bytes.IndexByte(data[offset:], '\n')
		flush := false
		stop := false
		if lineEnd == -1 || offset >= len(data) {
			stop = true
			offset = len(data)
			flush = true
			sectionBodyEnd = len(data)
		} else {
			line := data[offset : offset+lineEnd]
			if bytes.HasPrefix(line, []byte("##")) {
				colonPos := bytes.IndexByte(line, ':')
				flush = true
				sectionBodyEnd = offset
				if colonPos != -1 {
					nextSectionBodyBegin = offset + colonPos + 1
					nextSectionKey = string(bytes.TrimSpace(line[len("##"):colonPos]))
				} else {
					nextSectionBodyBegin = offset + lineEnd + 1
					nextSectionKey = string(line[len("##"):])
				}
			}
			offset += lineEnd + 1
		}
		if flush {
			if sectionKey != "" {
				if prefix != "" {
					sectionKey = prefix + "." + sectionKey
				}
				if !d.OverwriteAllowed {
					if _, ok := d.entries[sectionKey]; ok {
						return fmt.Errorf("%q key is already loaded", sectionKey)
					}
				}
				s := strings.TrimSpace(string(data[sectionBodyBegin:sectionBodyEnd]))
				s = strings.ReplaceAll(s, `\t`, d.tabSpaces)
				if d.replacer != nil {
					s = d.replacer.Replace(s)
				}
				d.entries[sectionKey] = s
			}
			sectionKey = nextSectionKey
			sectionBodyBegin = nextSectionBodyBegin
		}
		if stop {
			break
		}
	}

	return nil
}

func (d *Dictionary) Get2(keyParts ...string) (string, bool) {
	return d.get(d.entries, keyParts...)
}

func (d *Dictionary) Get(keyParts ...string) string {
	s, _ := d.get(d.entries, keyParts...)
	return s
}

func (d *Dictionary) Has(keyParts ...string) bool {
	_, ok := d.get(d.entries, keyParts...)
	return ok
}

func (d *Dictionary) WalkKeys(f func(k string)) {
	for k := range d.entries {
		f(k)
	}
}

func (d *Dictionary) Map(f func(k, v string) string) {
	for k, v := range d.entries {
		d.entries[k] = f(k, v)
	}
}

func (d *Dictionary) get(m map[string]string, keyParts ...string) (string, bool) {
	if len(keyParts) == 1 {
		return d.getSimple(d.entries, keyParts[0])
	}

	buf := d.keyBuf
	offset := 0
	for i, p := range keyParts {
		copy(buf[offset:], p)
		offset += len(p)
		if i != len(keyParts)-1 {
			buf[offset] = '.'
			offset++
		}
	}
	buf = buf[:offset]

	s, ok := m[string(buf)]
	if !ok {
		return "{{" + string(buf) + "}}", false
	}
	return s, true
}

func (d *Dictionary) getSimple(m map[string]string, key string) (string, bool) {
	s, ok := m[key]
	if !ok {
		return "{{" + key + "}}", false
	}
	return s, true
}
