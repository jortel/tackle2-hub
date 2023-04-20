package filter

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

const (
	QueryParam = "filter"
)

//
// New filter.
func New(ctx *gin.Context) (f Filter, err error) {
	p := Parser{}
	q := strings.Join(
		ctx.QueryArray(QueryParam),
		string(COMMA))
	f, err = p.Filter(q)
	return
}

//
// Filter is a collection of predicates.
type Filter struct {
	predicates []Predicate
}

//
// Field returns a field.
func (f *Filter) Field(name string) (field Field, found bool) {
	fields := f.Fields(name)
	if len(fields) > 0 {
		field = fields[0]
		found = true
	}
	return
}

//
// Fields returns fields.
func (f *Filter) Fields(name string) (fields []Field) {
	name = strings.ToLower(name)
	for _, p := range f.predicates {
		if strings.ToLower(p.Field.Value) == name {
			f := Field{p}
			fields = append(fields, f)
		}
	}
	return
}

//
// Resource returns a filter scoped to resource.
func (f *Filter) Resource(r string) (fr *Filter) {
	fr = &Filter{}
	var predicates []Predicate
	for _, p := range f.predicates {
		field := Field{p}
		if field.Resource() == r {
			p.Field.Value = field.Name()
			predicates = append(predicates, p)
		}
	}
	fr.predicates = predicates
	return
}

//
// Where applies the where clause.
func (f *Filter) Where(in *gorm.DB) (out *gorm.DB) {
	out = in
	for _, p := range f.predicates {
		field := Field{p}
		if field.Resource() == "" {
			out = out.Where(field.SQL())
		}
	}
	return
}

//
// Empty returns true when the filter has no predicates.
func (f *Filter) Empty() bool {
	return len(f.predicates) == 0
}

//
// Field predicate.
type Field struct {
	Predicate
}

//
// Name returns the field name.
func (f *Field) Name() (s string) {
	_, s = f.split()
	return
}

//
// As returns the renamed field.
func (f *Field) As(s string) (named Field) {
	named = Field{f.Predicate}
	named.Field.Value = s
	return
}

//
// Resource returns the field resource.
func (f *Field) Resource() (s string) {
	s, _ = f.split()
	return
}

//
// SQL builds SQL.
// Returns statement and value (for ?).
func (f *Field) SQL() (s string, v interface{}) {
	name := f.Name()
	switch len(f.Value) {
	case 0:
	case 1:
		switch f.Operator.Value {
		case string(LIKE):
			v = strings.Replace(f.Value[0].Value, "*", "%", -1)
			s = strings.Join(
				[]string{
					name,
					f.operator(),
					"?",
				},
				" ")
		default:
			v = AsValue(f.Value[0])
			s = strings.Join(
				[]string{
					name,
					f.operator(),
					"?",
				},
				" ")
		}
	default:
		operator := f.Value.ByKind(OPERATOR)[0]
		switch operator.Value[0] {
		case COMMA:
			// Unsupported.
		case OR:
			values := f.Value.ByKind(LITERAL, STR)
			collection := []interface{}{}
			for i := range values {
				v := AsValue(values[i])
				collection = append(collection, v)
			}
			v = collection
			s = strings.Join(
				[]string{
					name,
					f.operator(),
					"?",
				},
				" ")
		}
	}
	return
}

//
// Relation determines if the field is a relation.
func (f *Field) Relation() bool {
	operator := f.Value.ByKind(OPERATOR)
	return len(operator) > 0 && operator[0].Value[0] == COMMA
}

//
// split field name.
// format: resource.name
// The resource may be "" (anonymous).
func (f *Field) split() (relation string, name string) {
	part := strings.SplitN(f.Field.Value, ".", 2)
	if len(part) == 2 {
		relation = part[0]
		name = part[1]
	} else {
		name = part[0]
	}
	return
}

//
// operator returns SQL operator.
func (f *Field) operator() (s string) {
	switch len(f.Value) {
	case 1:
		s = f.Operator.Value
		switch s {
		case string(COLON):
			s = "="
		case string(LIKE):
			s = "LIKE"
		}
	default:
		switch len(f.Value) {
		case 0:
		case 1:
		}

		s = "IN"
	}

	return
}

//
// AsValue returns the real value.
func AsValue(t Token) (object interface{}) {
	v := t.Value
	object = v
	switch t.Kind {
	case LITERAL:
		n, err := strconv.Atoi(v)
		if err == nil {
			object = n
			break
		}
		b, err := strconv.ParseBool(v)
		if err == nil {
			object = b
			break
		}
	default:
	}
	return
}
