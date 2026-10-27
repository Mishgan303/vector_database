package storage

import "vectordb/internal/model"

func normalizeFilter(filter *model.Filter) ([]model.Condition, error) {
	if filter == nil {
		return nil, nil
	}
	conditions := make([]model.Condition, len(filter.Conditions))
	for i, c := range filter.Conditions {
		if c.Field == "" {
			return nil, invalidFilter("field must not be empty")
		}
		switch c.Operator {
		case model.FilterEQ:
			switch c.Value.(type) {
			case string, bool:
			default:
				n, ok := model.Number(c.Value)
				if !ok {
					return nil, invalidFilter("EQ value must be a scalar")
				}
				c.Value = n
			}
		case model.FilterGT, model.FilterGTE, model.FilterLT, model.FilterLTE:
			n, ok := model.Number(c.Value)
			if !ok {
				return nil, invalidFilter("ordered comparisons require numbers")
			}
			c.Value = n
		default:
			return nil, invalidFilter("unsupported filter operator")
		}
		conditions[i] = c
	}
	return conditions, nil
}

func matches(metadata map[string]any, conditions []model.Condition) bool {
	for _, c := range conditions {
		value := metadata[c.Field]
		if c.Operator == model.FilterEQ {
			if number, ok := model.Number(value); ok {
				value = number
			}
			if value != c.Value {
				return false
			}
			continue
		}
		number, ok := model.Number(value)
		if !ok {
			return false
		}
		want := c.Value.(float64)
		switch c.Operator {
		case model.FilterGT:
			if !(number > want) {
				return false
			}
		case model.FilterGTE:
			if !(number >= want) {
				return false
			}
		case model.FilterLT:
			if !(number < want) {
				return false
			}
		case model.FilterLTE:
			if !(number <= want) {
				return false
			}
		}
	}
	return true
}

func invalidFilter(message string) error {
	return model.DBError{Code: model.ErrorInvalidFilter, Message: message}
}
