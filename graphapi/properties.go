package graphapi

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
)

// Property is a node's input that can be a settable value
// Settable property types:
// "INT"			an int64
// "FLOAT"			a float64
// "STRING"			a single line, or multiline string
// "COMBO"			one of a given list of strings
// "BOOLEAN"		a labeled bool value
// "IMAGEUPLOAD"	image uploader
// "UNKNOWN"		everything else (unsettable)
type Property interface {
	TypeString() string
	Optional() bool
	Settable() bool
	Name() string
	SetTargetWidget(node *GraphNode, index int)
	GetTargetWidget() int
	GetTargetNode() *GraphNode
	GetValue() interface{}
	SetValue(v interface{}) error
	Serializable() bool
	SetSerializable(bool)
	AttachSecondaryProperty(p Property)
	Index() int

	UpdateParent(parent Property)
	ToIntProperty() (*IntProperty, bool)
	ToFloatProperty() (*FloatProperty, bool)
	ToBoolProperty() (*BoolProperty, bool)
	ToStringProperty() (*StringProperty, bool)
	ToComboProperty() (*ComboProperty, bool)
	ToImageUploadProperty() (*ImageUploadProperty, bool)
	ToUnknownProperty() (*UnknownProperty, bool)
	valueFromString(value string) interface{}
}

type BaseProperty struct {
	parent             Property
	name               string
	optional           bool
	target_node        *GraphNode
	target_value_index int
	serializable       bool
	secondaries        []Property
	override_property  interface{} // if non-nil, this value will be serialized
	index              int
}

func (b *BaseProperty) UpdateParent(parent Property) {
	b.parent = parent
}

func (b *BaseProperty) Serializable() bool {
	return b.serializable
}

func (b *BaseProperty) SetSerializable(val bool) {
	b.serializable = val
}

func (b *BaseProperty) AttachSecondaryProperty(p Property) {
	if b.secondaries == nil {
		b.secondaries = make([]Property, 0)
	}
	b.secondaries = append(b.secondaries, p)
}

func (b *BaseProperty) SetTargetWidget(node *GraphNode, index int) {
	b.target_node = node
	b.target_value_index = index
}

func (b *BaseProperty) GetTargetWidget() int {
	return b.target_value_index
}

func (b *BaseProperty) GetTargetNode() *GraphNode {
	return b.target_node
}

func (b *BaseProperty) GetValue() interface{} {
	if b.override_property != nil {
		return b.override_property
	}

	if b.target_node != nil {
		return b.target_node.WidgetValues[b.target_value_index]
	}
	return nil
}

// SetValue calls the protocol implementation for valueFromString to get
// the actual value that will be set.  valueFromString should perform
// conversion to it's native type and constrain it when needed
func (b *BaseProperty) SetValue(v interface{}) error {
	vs := fmt.Sprintf("%v", v)
	val := b.parent.valueFromString(vs)
	if val == nil {
		return errors.New("could not get converted type")
	}
	if b.target_node != nil {
		b.target_node.WidgetValues[b.target_value_index] = val
	} else {
		return errors.New("Property has no target node")
	}

	// if there are secondaries, set those too
	if b.secondaries != nil {
		for _, p := range b.secondaries {
			err := p.SetValue(val)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *BaseProperty) Index() int {
	return b.index
}

func (b *BaseProperty) ToIntProperty() (*IntProperty, bool) {
	if prop, ok := b.parent.(*IntProperty); ok {
		return prop, true
	}
	return nil, false
}
func (b *BaseProperty) ToFloatProperty() (*FloatProperty, bool) {
	if prop, ok := b.parent.(*FloatProperty); ok {
		return prop, true
	}
	return nil, false
}
func (b *BaseProperty) ToBoolProperty() (*BoolProperty, bool) {
	if prop, ok := b.parent.(*BoolProperty); ok {
		return prop, true
	}
	return nil, false
}
func (b *BaseProperty) ToStringProperty() (*StringProperty, bool) {
	if prop, ok := b.parent.(*StringProperty); ok {
		return prop, true
	}
	return nil, false
}
func (b *BaseProperty) ToComboProperty() (*ComboProperty, bool) {
	if prop, ok := b.parent.(*ComboProperty); ok {
		return prop, true
	}
	return nil, false
}
func (b *BaseProperty) ToImageUploadProperty() (*ImageUploadProperty, bool) {
	if prop, ok := b.parent.(*ImageUploadProperty); ok {
		return prop, true
	}
	return nil, false
}
func (b *BaseProperty) ToUnknownProperty() (*UnknownProperty, bool) {
	if prop, ok := b.parent.(*UnknownProperty); ok {
		return prop, true
	}
	return nil, false
}

type BoolProperty struct {
	BaseProperty
	Default  bool
	LabelOn  string
	LabelOff string
}

func newBoolProperty(input_name string, optional bool, data interface{}, index int) *Property {
	c := &BoolProperty{
		BaseProperty: BaseProperty{name: input_name, optional: optional, serializable: true, index: index},
		Default:      false,
	}
	c.parent = c

	if d, ok := data.(map[string]interface{}); ok {
		if val, ok := d["label_off"]; ok {
			c.LabelOn = val.(string)
		}

		if val, ok := d["label_off"]; ok {
			c.LabelOn = val.(string)
		}

		if val, ok := d["label_off"]; ok {
			c.Default = val.(bool)
		}
	}

	var retv Property = c
	return &retv
}
func (p *BoolProperty) TypeString() string {
	return "BOOLEAN"
}
func (p *BoolProperty) Optional() bool {
	return p.optional
}
func (p *BoolProperty) Settable() bool {
	return true
}
func (p *BoolProperty) Name() string {
	return p.name
}
func (p *BoolProperty) valueFromString(value string) interface{} {
	v, err := strconv.ParseBool(value)
	if err != nil {
		return nil
	}
	return v
}

type IntProperty struct {
	BaseProperty
	Default  int64
	Min      int64 // optional
	Max      int64 // optional
	Step     int64 // optional
	hasStep  bool
	hasRange bool
}

func newIntProperty(input_name string, optional bool, data interface{}, index int) *Property {
	c := &IntProperty{
		BaseProperty: BaseProperty{name: input_name, optional: optional, serializable: true, index: index},
		Default:      0,
		Min:          0,
		Max:          math.MaxInt64,
		Step:         0,
		hasRange:     false,
		hasStep:      false,
	}
	c.parent = Property(c)

	if d, ok := data.(map[string]interface{}); ok {
		// min?
		if val, ok := d["min"]; ok {
			c.Min = int64(val.(float64))
			c.hasRange = true
		}

		// max?
		if val, ok := d["max"]; ok {
			c.Max = int64(val.(float64))
			c.hasRange = true
		}

		// step?
		if val, ok := d["step"]; ok {
			c.Step = int64(val.(float64))
			c.hasStep = true
		}
	}

	var retv Property = c
	return &retv
}
func (p *IntProperty) TypeString() string {
	return "INT"
}
func (p *IntProperty) Optional() bool {
	return p.optional
}
func (p *IntProperty) HasStep() bool {
	return p.hasStep
}
func (p *IntProperty) HasRange() bool {
	return p.hasRange
}
func (p *IntProperty) Settable() bool {
	return true
}
func (p *IntProperty) Name() string {
	return p.name
}
func (p *IntProperty) valueFromString(value string) interface{} {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	if p.hasRange {
		v = int64(math.Min(float64(p.Max), float64(v)))
		v = int64(math.Max(float64(p.Min), float64(v)))
	}
	return v
}

type FloatProperty struct {
	BaseProperty
	Default  float64
	Min      float64
	Max      float64
	Step     float64
	hasStep  bool
	hasRange bool
}

func newFloatProperty(input_name string, optional bool, data interface{}, index int) *Property {
	c := &FloatProperty{
		BaseProperty: BaseProperty{name: input_name, optional: optional, serializable: true, index: index},
		Default:      0,
		Min:          0,
		Max:          math.MaxFloat64,
		Step:         0,
		hasStep:      false,
		hasRange:     false,
	}
	c.parent = c

	if d, ok := data.(map[string]interface{}); ok {
		// min?
		if val, ok := d["min"]; ok {
			c.Min = val.(float64)
			c.hasRange = true
		}

		// max?
		if val, ok := d["max"]; ok {
			c.Max = val.(float64)
			c.hasRange = true
		}

		// step?
		if val, ok := d["step"]; ok {
			c.Step = val.(float64)
			c.hasStep = true
		}
	}

	var retv Property = c
	return &retv
}
func (p *FloatProperty) TypeString() string {
	return "FLOAT"
}
func (p *FloatProperty) Optional() bool {
	return p.optional
}
func (p *FloatProperty) HasStep() bool {
	return p.hasStep
}
func (p *FloatProperty) HasRange() bool {
	return p.hasRange
}
func (p *FloatProperty) Settable() bool {
	return true
}
func (p *FloatProperty) Name() string {
	return p.name
}
func (p *FloatProperty) valueFromString(value string) interface{} {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	if p.hasRange {
		v = math.Min(v, p.Max)
		v = math.Max(v, p.Min)
	}
	return v
}

type StringProperty struct {
	BaseProperty
	Default   string
	Multiline bool
}

func newStringProperty(input_name string, optional bool, data interface{}, index int) *Property {
	c := &StringProperty{
		BaseProperty: BaseProperty{name: input_name, optional: optional, serializable: true, index: index},
		Default:      "",
		Multiline:    false,
	}
	c.parent = c

	if d, ok := data.(map[string]interface{}); ok {
		// default?
		if val, ok := d["default"]; ok {
			c.Default = val.(string)
		}

		// multiline?
		if val, ok := d["multiline"]; ok {
			c.Multiline = val.(bool)
		}
	}

	var retv Property = c
	return &retv
}
func (p *StringProperty) TypeString() string {
	return "STRING"
}
func (p *StringProperty) Optional() bool {
	return p.optional
}
func (p *StringProperty) Settable() bool {
	return true
}
func (p *StringProperty) Name() string {
	return p.name
}
func (p *StringProperty) valueFromString(value string) interface{} {
	return value
}

type ComboProperty struct {
	BaseProperty
	Values []string
}

func newComboProperty(input_name string, optional bool, input []interface{}, index int) *Property {
	c := &ComboProperty{
		BaseProperty: BaseProperty{name: input_name, optional: optional, serializable: true, index: index},
	}
	c.parent = c

	c.Values = make([]string, 0)
	for _, v := range input {
		if s, ok := v.(string); ok {
			c.Values = append(c.Values, s)
		}
	}
	var retv Property = c
	return &retv
}
func (p *ComboProperty) TypeString() string {
	return "COMBO"
}
func (p *ComboProperty) Optional() bool {
	return p.optional
}
func (p *ComboProperty) Settable() bool {
	return true
}
func (p *ComboProperty) Name() string {
	return p.name
}
func (p *ComboProperty) valueFromString(value string) interface{} {
	// ensure we have this string in our values
	for _, v := range p.Values {
		if value == v {
			return value
		}
	}
	return nil
}

// Append will add the new value to the combo if it's not already available, and then sets
// the target property to the given value
func (p *ComboProperty) Append(newValue string) {
	// do we already have this one?
	have := false
	for i := range p.Values {
		if p.Values[i] == newValue {
			// we have this
			have = true
			break
		}
	}
	if !have {
		p.Values = append(p.Values, newValue)
	}
	p.SetValue(newValue)
}

type ImageUploadProperty struct {
	BaseProperty
	TargetProperty *ComboProperty
}

func newImageUploadProperty(input_name string, target *ComboProperty, index int) *Property {
	c := &ImageUploadProperty{
		BaseProperty:   BaseProperty{name: input_name, optional: false, serializable: true, override_property: target.name, index: index},
		TargetProperty: target,
	}
	c.parent = c

	var retv Property = c
	return &retv
}
func (p *ImageUploadProperty) TypeString() string {
	return "IMAGEUPLOAD"
}
func (p *ImageUploadProperty) Optional() bool {
	return p.optional
}
func (p *ImageUploadProperty) Settable() bool {
	return false
}
func (p *ImageUploadProperty) Name() string {
	return p.name
}
func (p *ImageUploadProperty) SetFilename(filename string) {
	if p.TargetProperty != nil {
		p.TargetProperty.Append(filename)
	}
}
func (p *ImageUploadProperty) valueFromString(value string) interface{} {
	return nil
}

type UnknownProperty struct {
	BaseProperty
	TypeName string
}

func newUnknownProperty(input_name string, optional bool, typename string, index int) *Property {
	c := &UnknownProperty{
		BaseProperty: BaseProperty{name: input_name, optional: optional, serializable: true, index: index},
		TypeName:     typename,
	}
	c.parent = c

	var retv Property = c
	return &retv
}
func (p *UnknownProperty) TypeString() string {
	return p.TypeName
}
func (p *UnknownProperty) Optional() bool {
	return p.optional
}
func (p *UnknownProperty) Settable() bool {
	return false
}
func (p *UnknownProperty) Name() string {
	return p.name
}
func (p *UnknownProperty) valueFromString(value string) interface{} {
	return nil
}

func NewPropertyFromInput(input_name string, optional bool, input *interface{}, index int) *Property {
	// Convert the pointer back to an interface
	dereferenced := *input

	// Attempt to assert the interface as a slice of interfaces
	if slice, ok := dereferenced.([]interface{}); ok {
		// is it at least a size of 1?
		if len(slice) == 0 {
			return nil
		}

		// the first item is either an array of strings (a combo), or the property type
		if ptype, ok := slice[0].([]interface{}); ok {
			return newComboProperty(input_name, optional, ptype, index)
		} else {
			if stype, ok := slice[0].(string); ok {
				switch stype {
				case "STRING":
					return newStringProperty(input_name, optional, slice[1], index)
				case "INT":
					return newIntProperty(input_name, optional, slice[1], index)
				case "FLOAT":
					return newFloatProperty(input_name, optional, slice[1], index)
				case "BOOLEAN":
					return newBoolProperty(input_name, optional, stype, index)
				default:
					return newUnknownProperty(input_name, optional, stype, index)
				}
			}
		}
		log.Println("Success:", slice)
	}

	return nil
}
