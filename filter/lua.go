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
	Help:   `TBD`,
}

type LUAConfig struct {
	Script     string
	FilterName string
}

type LUA struct {
	l       *lua.LState
	luaFunc lua.LValue
}

func NewLUA(cfg baker.FilterParams) (baker.Filter, error) {
	dcfg := cfg.DecodedConfig.(*LUAConfig)

	l := lua.NewState()
	if err := l.DoFile(dcfg.Script); err != nil {
		return nil, fmt.Errorf("can't compile lua script %q: %v", dcfg.Script, err)
	}
	registerLUARecordType(l)
	// TODO: check function exists
	luaFunc := l.GetGlobal(dcfg.FilterName)

	f := &LUA{
		luaFunc: luaFunc,
		l:       l,
	}

	runtime.SetFinalizer(f, func(f *LUA) { f.l.Close() })

	return f, nil
}

func (t *LUA) Stats() baker.FilterStats { return baker.FilterStats{} }

func (t *LUA) Process(rec baker.Record, next func(baker.Record)) {
	luaNext := t.l.NewFunction(func(L *lua.LState) int {
		recordArg := fastcheckLuaRecord(L, 1)
		next(recordArg.r)
		return 0
	})

	err := t.l.CallByParam(lua.P{
		Fn:      t.luaFunc,
		NRet:    0,
		Protect: true,
	}, recordToLua(t.l, rec),
		luaNext)

	if err != nil {
		panic(err)
	}
}

const luaRecordTypeName = "record"

func registerLUARecordType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaRecordTypeName)
	L.SetGlobal(luaRecordTypeName, mt)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), luaRecordMethods))
}

func recordToLua(L *lua.LState, r baker.Record) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = &luaRecord{r: r}
	L.SetMetatable(ud, L.GetTypeMetatable(luaRecordTypeName))
	return ud
}

var luaRecordMethods = map[string]lua.LGFunction{
	"get": luaRecordGet,
	"set": luaRecordSet,
}

type luaRecord struct {
	r baker.Record
}

func checkLuaRecord(L *lua.LState, n int) *luaRecord {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*luaRecord); ok {
		return v
	}
	L.ArgError(n, fmt.Sprintf("record expected, got %#v", ud.Value))
	return nil
}

func fastcheckLuaRecord(L *lua.LState, n int) *luaRecord {
	return L.Get(n).(*lua.LUserData).Value.(*luaRecord)
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
