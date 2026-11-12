package plist

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// parsePlist decodes an XML or binary property list into Go values.
func Parse(data []byte) (any, error) {
	if bytes.HasPrefix(data, []byte("bplist00")) {
		return parseBinaryPlist(data)
	}
	return parseXMLPlist(data)
}

type xmlNode struct {
	XMLName xml.Name
	Content string    `xml:",chardata"`
	Nodes   []xmlNode `xml:",any"`
}

func parseXMLPlist(data []byte) (any, error) {
	var root xmlNode
	decoder := xml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	if root.XMLName.Local != "plist" || len(root.Nodes) == 0 {
		return nil, fmt.Errorf("plist inválido")
	}
	return xmlValue(root.Nodes[0])
}

func xmlValue(node xmlNode) (any, error) {
	switch node.XMLName.Local {
	case "dict":
		result := map[string]any{}
		var key *string
		for _, child := range node.Nodes {
			if child.XMLName.Local == "key" {
				value := child.Content
				key = &value
				continue
			}
			if key == nil {
				continue
			}
			value, err := xmlValue(child)
			if err != nil {
				return nil, err
			}
			result[*key] = value
			key = nil
		}
		return result, nil
	case "array":
		result := make([]any, 0, len(node.Nodes))
		for _, child := range node.Nodes {
			value, err := xmlValue(child)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case "string":
		return node.Content, nil
	case "integer":
		value, err := strconv.ParseInt(strings.TrimSpace(node.Content), 0, 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	case "real":
		value, err := strconv.ParseFloat(strings.TrimSpace(node.Content), 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "date":
		value, err := time.Parse(time.RFC3339, strings.TrimSpace(node.Content))
		if err != nil {
			return nil, err
		}
		return value, nil
	case "data":
		return node.Content, nil
	default:
		return nil, fmt.Errorf("tipo de plist no soportado: %s", node.XMLName.Local)
	}
}

type binaryPlist struct {
	data          []byte
	offsets       []uint64
	objectRefSize int
}

func parseBinaryPlist(data []byte) (any, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("plist binario truncado")
	}
	trailer := data[len(data)-32:]
	offsetIntSize := int(trailer[6])
	objectRefSize := int(trailer[7])
	numObjects := binary.BigEndian.Uint64(trailer[8:16])
	topObject := binary.BigEndian.Uint64(trailer[16:24])
	offsetTableOffset := binary.BigEndian.Uint64(trailer[24:32])
	if offsetIntSize < 1 || offsetIntSize > 8 || objectRefSize < 1 || objectRefSize > 8 {
		return nil, fmt.Errorf("plist binario con tamaños inválidos")
	}
	if numObjects == 0 || numObjects > uint64(len(data)) {
		return nil, fmt.Errorf("plist binario con objetos inválidos")
	}
	if offsetTableOffset < 8 || offsetTableOffset+numObjects*uint64(offsetIntSize) > uint64(len(data)) {
		return nil, fmt.Errorf("plist binario truncado")
	}
	offsets := make([]uint64, numObjects)
	for index := uint64(0); index < numObjects; index++ {
		start := offsetTableOffset + index*uint64(offsetIntSize)
		offsets[index] = readUint(data[start : start+uint64(offsetIntSize)])
	}
	if topObject >= numObjects {
		return nil, fmt.Errorf("objeto raíz inválido")
	}
	parser := &binaryPlist{data: data, offsets: offsets, objectRefSize: objectRefSize}
	return parser.object(topObject, 0)
}

func readUint(data []byte) uint64 {
	var value uint64
	for _, b := range data {
		value = value<<8 | uint64(b)
	}
	return value
}

func (p *binaryPlist) object(reference uint64, depth int) (any, error) {
	if depth > 128 {
		return nil, fmt.Errorf("plist binario demasiado profundo")
	}
	if reference >= uint64(len(p.offsets)) {
		return nil, fmt.Errorf("referencia de plist fuera de rango")
	}
	offset := p.offsets[reference]
	if offset >= uint64(len(p.data)) {
		return nil, fmt.Errorf("objeto de plist fuera de rango")
	}
	marker := p.data[offset]
	kind := marker >> 4
	info := marker & 0x0f
	pos := offset + 1
	switch kind {
	case 0x0:
		if info == 0x8 {
			return false, nil
		}
		if info == 0x9 {
			return true, nil
		}
		return nil, nil
	case 0x1:
		size := uint64(1) << info
		if pos+size > uint64(len(p.data)) {
			return nil, fmt.Errorf("entero truncado")
		}
		raw := p.data[pos : pos+size]
		if size <= 8 {
			return int64(readUint(raw)), nil
		}
		return int64(binary.BigEndian.Uint64(raw[len(raw)-8:])), nil
	case 0x2:
		size := uint64(1) << info
		if pos+size > uint64(len(p.data)) {
			return nil, fmt.Errorf("real truncado")
		}
		if size == 4 {
			return float64(math.Float32frombits(binary.BigEndian.Uint32(p.data[pos:]))), nil
		}
		return math.Float64frombits(binary.BigEndian.Uint64(p.data[pos:])), nil
	case 0x4:
		length, next, err := p.length(info, pos)
		if err != nil || next+length > uint64(len(p.data)) {
			return nil, fmt.Errorf("datos truncados")
		}
		return p.data[next : next+length], nil
	case 0x5:
		length, next, err := p.length(info, pos)
		if err != nil || next+length > uint64(len(p.data)) {
			return nil, fmt.Errorf("cadena truncada")
		}
		return string(p.data[next : next+length]), nil
	case 0x6:
		length, next, err := p.length(info, pos)
		if err != nil || next+length*2 > uint64(len(p.data)) {
			return nil, fmt.Errorf("cadena UTF-16 truncada")
		}
		runes := make([]rune, 0, length)
		for index := uint64(0); index < length; index++ {
			value := binary.BigEndian.Uint16(p.data[next+index*2:])
			if value >= 0xD800 && value <= 0xDBFF && index+1 < length {
				low := binary.BigEndian.Uint16(p.data[next+(index+1)*2:])
				if low >= 0xDC00 && low <= 0xDFFF {
					runes = append(runes, rune(0x10000+(uint32(value-0xD800)<<10)+uint32(low-0xDC00)))
					index++
					continue
				}
			}
			runes = append(runes, rune(value))
		}
		return string(runes), nil
	case 0x8:
		if pos+uint64(info)+1 > uint64(len(p.data)) {
			return nil, fmt.Errorf("UID truncado")
		}
		return readUint(p.data[pos : pos+uint64(info)+1]), nil
	case 0xA, 0xC:
		length, next, err := p.length(info, pos)
		if err != nil {
			return nil, err
		}
		result := make([]any, 0, length)
		for index := uint64(0); index < length; index++ {
			start := next + index*uint64(p.objectRefSize)
			if start+uint64(p.objectRefSize) > uint64(len(p.data)) {
				return nil, fmt.Errorf("array truncado")
			}
			ref := readUint(p.data[start : start+uint64(p.objectRefSize)])
			value, err := p.object(ref, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case 0xD:
		length, next, err := p.length(info, pos)
		if err != nil {
			return nil, err
		}
		result := make(map[string]any, length)
		for index := uint64(0); index < length; index++ {
			keyStart := next + index*uint64(p.objectRefSize)
			valueStart := next + (length+index)*uint64(p.objectRefSize)
			if keyStart+uint64(p.objectRefSize) > uint64(len(p.data)) || valueStart+uint64(p.objectRefSize) > uint64(len(p.data)) {
				return nil, fmt.Errorf("diccionario truncado")
			}
			keyRef := readUint(p.data[keyStart : keyStart+uint64(p.objectRefSize)])
			valueRef := readUint(p.data[valueStart : valueStart+uint64(p.objectRefSize)])
			keyValue, err := p.object(keyRef, depth+1)
			if err != nil {
				return nil, err
			}
			key, ok := keyValue.(string)
			if !ok {
				return nil, fmt.Errorf("clave de plist no textual")
			}
			value, err := p.object(valueRef, depth+1)
			if err != nil {
				return nil, err
			}
			result[key] = value
		}
		return result, nil
	default:
		return nil, fmt.Errorf("tipo de plist binario no soportado: 0x%x", marker)
	}
}

func (p *binaryPlist) length(info byte, pos uint64) (uint64, uint64, error) {
	if info != 0x0f {
		return uint64(info), pos, nil
	}
	if pos >= uint64(len(p.data)) {
		return 0, 0, fmt.Errorf("longitud truncada")
	}
	marker := p.data[pos]
	if marker>>4 != 0x1 {
		return 0, 0, fmt.Errorf("longitud no entera")
	}
	size := uint64(1) << (marker & 0x0f)
	if pos+1+size > uint64(len(p.data)) {
		return 0, 0, fmt.Errorf("longitud truncada")
	}
	return readUint(p.data[pos+1 : pos+1+size]), pos + 1 + size, nil
}

// DuplicateKey returns the first duplicated key inside an XML plist dict.
func DuplicateKey(data []byte) string {
	if bytes.HasPrefix(data, []byte("bplist00")) {
		return ""
	}
	var root xmlNode
	if err := xml.Unmarshal(data, &root); err != nil {
		return ""
	}
	return duplicateInNode(root)
}

func duplicateInNode(node xmlNode) string {
	if node.XMLName.Local == "dict" {
		seen := map[string]bool{}
		for _, child := range node.Nodes {
			if child.XMLName.Local != "key" {
				continue
			}
			key := child.Content
			if seen[key] {
				return key
			}
			seen[key] = true
		}
	}
	for _, child := range node.Nodes {
		if duplicate := duplicateInNode(child); duplicate != "" {
			return duplicate
		}
	}
	return ""
}
