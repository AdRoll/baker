package filter

import (
	"fmt"
	"runtime"

	"github.com/AdRoll/baker"

	lua "github.com/yuin/gopher-lua"
)

// LUADesc describes the LUA filter
var LUADesc = baker.FilterDesc{
	Name:   "LUA",
	New:    NewLUA,
	Config: &LUAConfig{},
	Help:   `Run a baker filter defined in a lua script`,
}

// LUAConfig holds the configuration for the LUA filter.
type LUAConfig struct {
	Script     string `help:"Path to the lua script where the baker filter is defined" required:"true"`
	FilterName string `help:"Name of the lua function to run as baker filter" required:"true"`
}

// LUA allows to run a baker filter from an external script written in lua.
type LUA struct {
	l       *lua.LState    // lua state used during all the baker filter lifetime
	ud      *lua.LUserData // pre-allocated (reused) userdata for the processed record
	luaFunc lua.LValue     // lua filter function
	luaNext *lua.LFunction // lua next function (reused)
	next    func(baker.Record)
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

	// Preallocate the userdata we use to wrap the record passed to the filter.
	// We can do this since a single instance of a baker filter is only ever
	// processing a single record at a time, so we can reuse the lua userdata
	// structure for it. This reduces allocations.
	ud := l.NewUserData()
	l.SetMetatable(ud, l.GetTypeMetatable(luaRecordTypeName))

	f := &LUA{}

	// Preallocate the lua next function passed to the filter.
	luaNext := l.NewFunction(func(L *lua.LState) int {
		f.next(fastcheckLuaRecord(L, 1).r)
		return 0
	})

	f.l = l
	f.ud = ud
	f.luaNext = luaNext
	f.luaFunc = luaFunc

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

// TODO: at the moment LUA filter doesn't publish stats.
// There are multiple ways to do it, either require the filter to update
// the numbers of processed and filtered records, or deduce them automatically
// by hooking into next and Process functions (if that proves too costly
// this could be disabled in configuration.
func (t *LUA) Stats() baker.FilterStats { return baker.FilterStats{} }

// Process forwards records to the lua-written filter.
func (t *LUA) Process(rec baker.Record, next func(baker.Record)) {
	// Modify the record inside the pre-allocated user value
	t.ud.Value = &luaRecord{r: rec}

	// Set the next function which is called by the lua filter to the one
	// we just received.
	t.next = next

	err := t.l.CallByParam(lua.P{
		Fn:      t.luaFunc,
		NRet:    0,
		Protect: true,
	}, t.ud, t.luaNext)

	// TODO: should not panic here and instead increment a filter-specific
	// metric that tracks the number of lua runtime errors.
	if err != nil {
		panic(err)
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
