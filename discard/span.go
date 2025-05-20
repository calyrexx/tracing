package discard

type span struct{}

func (s span) End() {}

func (s span) SetStringAttribute(key, value string) {}

func (s span) SetIntAttribute(key string, value int) {}

func (s span) SetBoolAttribute(key string, value bool) {}

func (s span) SetJSONAttribute(key string, value interface{}) {}

func (s span) AddEvent(name string) {}

func (s span) AddEventWithInt(name string, key string, value int) {}

func (s span) AddEventWithBool(name string, key string, value bool) {}

func (s span) AddEventWithString(name string, key string, value string) {}

func (s span) RecordError(err error) {}
