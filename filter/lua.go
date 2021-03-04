package filter

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/AdRoll/baker"

	lua "github.com/yuin/gopher-lua"
)

var luaHelp = `
This filter runs a baker filter defined in a LUA script.
It's useful to quickly write and run a Baker filter without having to recompile Baker.

This filter is based on [GopherLua](github.com/yuin/gopher-lua), which is an LUA5.1 virtual machine.

To use this filter you need to declare a function in an lua file. This function serves the same purpose
of the equivalent ` + ticks("baker.Filter.Process") + ` method one would write for a Go filter.

This is a simple filter which writes "hey" to the field "f0" of every record: 
` + startBlock("lua") + `
-- rec is a record object to be processed
-- next is the function next(record) to forward, if you want to, the record into the filter chain.
function myFilter(rec, next)
    rec:set(fieldNames["f0"], "hey")
    next(rec)
end
` + endBlock() + `

### The ` + ticks("Record") + ` table

The first argument received by your lua filter function is a Record table. The Record table is an surrogate 
for its Go counterpart, ` + ticks("baker.Record") + `. The Lua Record table defines the following methods:

#### get

- Go: ` + ticks("Record.Get(FieldIndex) []byte") + `
- lua: ` + ticks("record:get(idx) -> string") + `

Same as its Go counterpart ` + ticks("record::get") + ` takes a integer representing the field index and returns its value.
But, unlike in Go where the value is a ` + ticks("[]byte]") + `, in lua it's returned as a string, to allow for fast protoyping.

#### "set"

- Go: ` + ticks("Record.Set(FieldIndex, []byte)") + `
- lua: ` + ticks("record:set(idx, string)") + `

Same as its Go counterpart ` + ticks("record::set") + ` takes a integer representing the field index and the value to set.
But, unlike in Go where the value is a ` + ticks("[]byte]") + `, in lua it's a string, to allow for fast protoyping.

#### "copy"

- Go: ` + ticks("Record.Copy() Record") + `
- lua: ` + ticks("record:copy() -> record") + `

Calling ` + ticks("record::copy") + ` returns a new record, a deep-copy of the original.

#### "clear"

- Go: ` + ticks("Record.Clear()") + `
- lua: ` + ticks("lua: record:clear()") + `

Calling ` + ticks("record::clear") + ` clears the records internal state, making all its fields empty.


### Global functions

#### createRecord

- Go: ` + ticks("Components.CreateRecord() Record") + `
- lua: ` + ticks("createRecord -> record") + `

` + ticks("createRecord") + ` is the lua equivalent of the ` + ticks("CreateRecord") + ` function passed to your filter during construction.
It allows to create a new Record instance.

#### validateRecord

- Go: ` + ticks("Components.ValidateRecord(Record) (bool, FieldIndex)") + `
- lua: ` + ticks("validateRecord(record) -> (bool, int)") + `

` + ticks("validateRecord") + ` is the lua equivalent of the ` + ticks("ValidateRecord") + ` function passed to your filter during construction.
It validates a given record with respect to the validation function, returning a boolean indicating whether 
the record is a valid one, if false, the returned integer indicates the index of the first invalid field it met.

#### fieldByName

- Go: ` + ticks("Components.FieldByName(string) (FieldIndex, bool)") + `
- lua: ` + ticks("fieldByName(string) -> int|nil") + `

` + ticks("fieldByName") + ` is the lua equivalent of the ` + ticks("FieldByName") + ` function passed to your filter during construction.
It allows to lookup a field index by its name, returning the index or nil if no field exists with this index.

### Global tables

#### fieldNames

- Go: ` + ticks("Components.FieldNames []string") + `
- lua: ` + ticks("fieldNames") + `

` + ticks("fieldNames") + ` is an integed-indexed table, in other words an array, containing all field names, as ` + ticks("FieldNames") + ` in Go.
`

// TODO(arl): ideally this functions should not be required, but writing
// markdown documentation in Go strings is tedious and error-prone.

func ticks(s string) string         { return "`" + s + "`" }
func startBlock(lang string) string { return "```" + lang }
func endBlock() string              { return "```" }

// LUADesc describes the LUA filter
var LUADesc = baker.FilterDesc{
	Name:   "LUA",
	New:    NewLUA,
	Config: &LUAConfig{},
	Help:   luaHelp,
}

// LUAConfig holds the configuration for the LUA filter.
type LUAConfig struct {
	Script     string `help:"Path to the lua script where the baker filter is defined" required:"true"`
	FilterName string `help:"Name of the lua function to run as baker filter" required:"true"`
}

// LUA allows to run a baker filter from an external script written in lua.
type LUA struct {
	mu sync.Mutex
	l  *lua.LState // lua state used during all the baker filter lifetime

	luaFunc  lua.LValue // lua filter function
	recordMt lua.LValue

	nprocessed, nfiltered int64
}

// NewLUA returns a new LUA filter.
func NewLUA(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*LUAConfig)

	l := lua.NewState()
	if err := l.DoFile(dcfg.Script); err != nil {
		return nil, fmt.Errorf("can't compile lua script %q: %v", dcfg.Script, err)
	}

	registerLUATypes(l, cfg.ComponentParams)

	luaFunc := l.GetGlobal(dcfg.FilterName)
	if luaFunc.Type() == lua.LTNil {
		return nil, fmt.Errorf("can't find lua filter %q in script %q", dcfg.FilterName, dcfg.Script)
	}

	f := &LUA{
		l:        l,
		recordMt: l.GetTypeMetatable(luaRecordTypeName),
		luaFunc:  luaFunc,
	}

	// Since a filter has no way to know when it's deallocated we set a
	// finaliser on the lua state instance, which gives us the occasion to close
	// it.
	runtime.SetFinalizer(f, func(f *LUA) { f.l.Close() })

	return f, nil
}

// registerLUATypes registers, in the given lua state, some lua types
// and utility functions useful to run a baker filter:
//  - the record type
//  - createRecord function (creates and returns a new record)
//  - validateRecord function (takes a record and returns a boolean and a
//    number), see baker.ComponentParams.ValidateRecord
//  - fieldByName function (returns a field index given its name, or nil
//    if the given field name doesn't exist)
//  - fieldNames, an lua table where field names are indexed by their field
//    field indexes
func registerLUATypes(l *lua.LState, comp baker.ComponentParams) {
	registerLUARecordType(l)

	// Registers the 'createRecord' lua function.
	l.SetGlobal("createRecord", l.NewFunction(func(L *lua.LState) int {
		rec := comp.CreateRecord()
		ud := recordToLua(l, rec)
		L.Push(ud)
		return 1
	}))

	l.SetGlobal("validateRecord", l.NewFunction(func(L *lua.LState) int {
		luar := fastcheckLuaRecord(l, 1)
		ok, fidx := comp.ValidateRecord(luar.r)
		l.Push(lua.LBool(ok))
		l.Push(lua.LNumber(fidx))
		return 2
	}))

	l.SetGlobal("fieldByName", l.NewFunction(func(L *lua.LState) int {
		fname := L.CheckString(1)
		fidx, ok := comp.FieldByName(fname)
		if !ok {
			l.Push(lua.LNil)
		} else {
			l.Push(lua.LNumber(fidx))
		}
		return 1
	}))

	// Create the fieldNaames table.
	fields := l.NewTable()
	for fidx, fname := range comp.FieldNames {
		fields.RawSet(lua.LNumber(fidx), lua.LString(fname))
	}
	l.SetGlobal("fieldNames", fields)
}

func (t *LUA) Stats() baker.FilterStats {
	return baker.FilterStats{
		NumProcessedLines: atomic.LoadInt64(&t.nprocessed),
		NumFilteredLines:  atomic.LoadInt64(&t.nfiltered),
	}
}

// Process forwards records to the lua-written filter.
func (t *LUA) Process(rec baker.Record, next func(baker.Record)) {
	atomic.AddInt64(&t.nprocessed, 1)

	t.mu.Lock()
	defer t.mu.Unlock()

	nextCalled := false
	luaNext := t.l.NewFunction(func(L *lua.LState) int {
		next(fastcheckLuaRecord(L, 1).r)
		nextCalled = true
		return 0
	})

	// Wrap the incoming record into an lua user data having the 'record' type meta-table.
	ud := t.l.NewUserData()
	t.l.SetMetatable(ud, t.recordMt)
	ud.Value = &luaRecord{r: rec}

	err := t.l.CallByParam(lua.P{
		Fn:      t.luaFunc,
		NRet:    0,
		Protect: true,
	}, ud, luaNext)

	// TODO: should not panic here and instead increment a filter-specific
	// metric that tracks the number of lua runtime errors.
	if err != nil {
		panic(err)
	}

	if !nextCalled {
		atomic.AddInt64(&t.nfiltered, 1)
	}
}

// lua record methods

const luaRecordTypeName = "record"

// registers the 'record' type into the given lua state.
func registerLUARecordType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaRecordTypeName)
	L.SetGlobal(luaRecordTypeName, mt)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), luaRecordMethods))
}

// converts a baker.Record to lua user data, suitable to be pushed onto
// an lua stack.
func recordToLua(L *lua.LState, r baker.Record) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = &luaRecord{r: r}
	// TODO(arl) record type metatable likely never changes so we could avoid
	// the extra lookup here
	L.SetMetatable(ud, L.GetTypeMetatable(luaRecordTypeName))
	return ud
}

// checks that the element at current stack index n is an lua
// user data, holding an luaRecord, and returns it. Raises an lua
// runtime error if the element is not a record.
func checkLuaRecord(L *lua.LState, n int) *luaRecord {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*luaRecord); ok {
		return v
	}
	L.ArgError(n, fmt.Sprintf("record expected, got %#v", ud.Value))
	return nil
}

// faster version of checkLuaRecord that panics if the stack element
// is not a record, rather than raising an lua runtime error.
func fastcheckLuaRecord(L *lua.LState, n int) *luaRecord {
	return L.Get(n).(*lua.LUserData).Value.(*luaRecord)
}

// holds the lua-bound methods of baker.Record.
var luaRecordMethods = map[string]lua.LGFunction{
	"get":   luaRecordGet,
	"set":   luaRecordSet,
	"copy":  luaRecordCopy,
	"clear": luaRecordClear,
}

// lua wrapper over a baker Record.
type luaRecord struct {
	r baker.Record
}

// record:get(int) returns string
func luaRecordGet(L *lua.LState) int {
	luar := fastcheckLuaRecord(L, 1)
	fidx := L.CheckInt(2)

	buf := luar.r.Get(baker.FieldIndex(fidx))

	L.Push(lua.LString(string(buf)))
	return 1
}

// record:set(int, string)
func luaRecordSet(L *lua.LState) int {
	luar := fastcheckLuaRecord(L, 1)
	fidx := L.CheckInt(2)
	val := L.CheckString(3)

	luar.r.Set(baker.FieldIndex(fidx), []byte(val))

	return 0
}

// record:copy() record
func luaRecordCopy(L *lua.LState) int {
	luar := fastcheckLuaRecord(L, 1)

	cpy := luar.r.Copy()
	ud := recordToLua(L, cpy)
	L.Push(ud)

	return 1
}

// record:clear()
func luaRecordClear(L *lua.LState) int {
	luar := fastcheckLuaRecord(L, 1)
	luar.r.Clear()

	return 0
}
