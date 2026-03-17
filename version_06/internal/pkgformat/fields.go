package pkgformat

import "fwparse/internal/model"

func TryParsePkgFieldsBE(data []byte, start int, maxFields int) []model.PkgField {
	var fields []model.PkgField
	pos := start

	for pos+3 <= len(data) && len(fields) < maxFields {
		tag := data[pos]
		length := int(data[pos+1])<<8 | int(data[pos+2])

		if length <= 0 || length > 512 {
			break
		}

		valueStart := pos + 3
		valueEnd := valueStart + length
		if valueEnd > len(data) {
			break
		}

		fields = append(fields, model.PkgField{
			Offset: pos,
			Tag:    tag,
			Length: length,
			Value:  data[valueStart:valueEnd],
		})

		pos = valueEnd
	}

	return fields
}

func TryParsePkgFieldsLE(data []byte, start int, maxFields int) []model.PkgField {
	var fields []model.PkgField
	pos := start

	for pos+3 <= len(data) && len(fields) < maxFields {
		tag := data[pos]
		length := int(data[pos+1]) | int(data[pos+2])<<8

		if length <= 0 || length > 512 {
			break
		}

		valueStart := pos + 3
		valueEnd := valueStart + length
		if valueEnd > len(data) {
			break
		}

		fields = append(fields, model.PkgField{
			Offset: pos,
			Tag:    tag,
			Length: length,
			Value:  data[valueStart:valueEnd],
		})

		pos = valueEnd
	}

	return fields
}

func scorePreview(value []byte) bool {
	for _, b := range value {
		if b >= 32 && b <= 126 {
			return true
		}
	}
	return false
}

func ScorePkgFields(fields []model.PkgField) int {
	score := 0

	for _, f := range fields {
		score += 1

		if scorePreview(f.Value) {
			score += 2
		}

		if f.Length > 8 {
			score += 1
		}
	}

	return score
}

func FindBestPkgFieldStart(data []byte) (int, string, []model.PkgField) {
	limit := 64
	if len(data) < limit {
		limit = len(data)
	}

	bestStart := -1
	bestMode := ""
	var bestFields []model.PkgField
	bestScore := -1

	for start := 0; start < limit; start++ {
		fieldsBE := TryParsePkgFieldsBE(data, start, 12)
		scoreBE := ScorePkgFields(fieldsBE)

		if len(fieldsBE) >= 2 && scoreBE > bestScore {
			bestStart = start
			bestMode = "big-endian"
			bestFields = fieldsBE
			bestScore = scoreBE
		}

		fieldsLE := TryParsePkgFieldsLE(data, start, 12)
		scoreLE := ScorePkgFields(fieldsLE)

		if len(fieldsLE) >= 2 && scoreLE > bestScore {
			bestStart = start
			bestMode = "little-endian"
			bestFields = fieldsLE
			bestScore = scoreLE
		}
	}

	return bestStart, bestMode, bestFields
}
