package filter

import "math"

//
// Parser used to parse the filter.
type Parser struct {
}

//
// Filter parses the filter and builds a Filter.
func (r *Parser) Filter(filter string) (f Filter, err error) {
	if filter == "" {
		return
	}
	lexer := Lexer{}
	err = lexer.With(string(COMMA) + filter)
	if err != nil {
		return
	}
	var bfr []Token
	for {
		token, next := lexer.next()
		if !next {
			break
		}
		if len(bfr) > 2 {
			if bfr[0].Kind != OPERATOR || bfr[2].Kind != OPERATOR {
				err = &BadFilterError{"Syntax error."}
				return
			}
			switch token.Kind {
			case LITERAL, STR:
				p := Predicate{
					Unused:   bfr[0],
					Field:    bfr[1],
					Operator: bfr[2],
					Value:    Value{token},
				}
				f.predicates = append(f.predicates, p)
				bfr = nil
			case LPAREN:
				lexer.put()
				list := List{&lexer}
				v, nErr := list.Build()
				if nErr != nil {
					err = nErr
					return
				}
				p := Predicate{
					Unused:   bfr[0],
					Field:    bfr[1],
					Operator: bfr[2],
					Value:    v,
				}
				f.predicates = append(f.predicates, p)
				bfr = nil
			}
		} else {
			bfr = append(bfr, token)
		}
	}
	if len(bfr) != 0 {
		err = &BadFilterError{"Syntax error."}
		return
	}
	return
}

//
// Predicate filter predicate.
type Predicate struct {
	Unused   Token
	Field    Token
	Operator Token
	Value    Value
}

//
// Value term value.
type Value []Token

//
// ByKind returns values by kind.
func (r Value) ByKind(kind ...byte) (matched []Token) {
	for _, t := range r {
		for _, k := range kind {
			if t.Kind == k {
				matched = append(matched, t)
			}
		}
	}
	return
}

//
// List construct.
// Example: (red|blue|green)
type List struct {
	*Lexer
}

//
// Build the value.
func (r *List) Build() (v Value, err error) {
	for {
		token, next := r.next()
		if !next {
			err = &BadFilterError{"End ')' not found."}
			break
		}
		switch token.Kind {
		case LITERAL, STR:
			v = append(v, token)
		case OPERATOR:
			switch token.Value {
			case string(COMMA),
				string(OR):
				v = append(v, token)
			default:
				err = &BadFilterError{
					"List: separator must be `,` `|`"}
				return
			}
		case LPAREN:
			// ignored.
		case RPAREN:
			err = r.validate(v)
			return
		default:
			err = &BadFilterError{
				"List: " + token.Value + " not expected.",
			}
			return
		}
	}

	return
}

//
// validate the result.
func (r *List) validate(v Value) (err error) {
	lastOp := byte(0)
	for i := range v {
		if math.Mod(float64(i), 2) == 0 {
			switch v[i].Kind {
			case LITERAL,
				STR:
			default:
				err = &BadFilterError{
					"List: (LITERAL|STR) expected."}
				return
			}
		} else {
			switch v[i].Kind {
			case OPERATOR:
				operator := v[i].Value[0]
				if lastOp != 0 {
					if operator != lastOp {
						err = &BadFilterError{
							"List: Mixed operator detected."}
						return
					}
				}
				lastOp = operator
			default:
				err = &BadFilterError{
					"List: OPERATOR expected."}
				return
			}
		}
	}
	if len(v) == 0 {
		err = &BadFilterError{"List: Empty."}
	}
	return
}
