package bflags

import (
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
)

func scanStructFields(u interface{}) (string, []reflect.StructField) {
	if reflect.TypeOf(u).Kind() != reflect.Ptr ||
		reflect.ValueOf(u).Kind() != reflect.Ptr ||
		reflect.ValueOf(u).Elem().Kind() != reflect.Struct {
		// support only structs
		return "", nil
	}
	val := reflect.ValueOf(u).Elem()
	typ := val.Type()
	typeName := typ.Name()
	result := make([]reflect.StructField, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		result[i] = typ.Field(i)
		//fmt.Println("field", i, result[i].Name)
	}
	return typeName, result
}

type SimpleTaggedStruct struct {
	Stringval string `cmd:"flag"`
	Ips       net.IP `cmd:"flag,ips,peer ips,i,true,true"`
	Ip        net.IP `cmd:"arg,id,node ip,2"`
	QIp       net.IP `json:"qip,omitempty"`
	Ignore    string
}

func newSimpleTaggedStruct() *SimpleTaggedStruct {
	return &SimpleTaggedStruct{
		Stringval: "v",
		QIp:       net.IP{192, 168, 0, 0},
	}
}

func TestParseTaggedFields(t *testing.T) {
	sts := newSimpleTaggedStruct()
	name, fields := scanStructFields(sts)
	require.Equal(t, "SimpleTaggedStruct", name)
	require.Equal(t, 5, len(fields))

	spec := parseFieldTag(fields[0])
	require.NotNil(t, spec)
	//fmt.Printf("%#v\n", spec)
	require.Equal(t,
		&flagSpec{
			name:        "Stringval",
			description: "",
			shorthand:   "",
			persistent:  false,
			required:    false},
		spec)

	spec = parseFieldTag(fields[1])
	require.NotNil(t, spec)
	//fmt.Printf("%#v\n", spec)
	require.Equal(t,
		&flagSpec{
			name:        "ips",
			description: "peer ips",
			shorthand:   "i",
			persistent:  true,
			required:    true},
		spec)

	spec = parseFieldTag(fields[2])
	require.NotNil(t, spec)
	//fmt.Printf("%#v\n", spec)
	require.Equal(t,
		&argSpec{
			name:        "id",
			description: "node ip",
			order:       2,
		},
		spec)

	spec = parseFieldTag(fields[3])
	require.Nil(t, spec)
	spec = parseFieldTag(fields[4])
	require.Nil(t, spec)
}

func assertFlag(t *testing.T, c *cobra.Command, name string) *pflag.Flag {
	flags, err := GetCmdFlagSet(c)
	require.NoError(t, err)
	_, ok := flags[cmdFlag(name)]
	require.True(t, ok)
	pf := c.Flag(name)
	require.NotNil(t, pf)
	return pf
}

type TestPtrBoolIntString struct {
	Is     *bool   `cmd:"flag" meta:"ok"`
	Int    *int    `cmd:"flag" meta:"one,two"`
	Int64  *int64  `cmd:"flag"`
	Str    *string `cmd:"flag"`
	Arg    *string `cmd:"arg" meta:"ok"`
	Ignore string
}

func TestAnnotations(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestPtrBoolIntString{}
	err := Bind(c, sts)
	require.NoError(t, err)

	flags, err := GetCmdFlagSet(c)
	require.NoError(t, err)
	f, ok := flags[cmdFlag("Is")]
	require.True(t, ok)
	require.Equal(t, 1, len(f.Annotations))
	require.Equal(t, "ok", f.Annotations[0])

	f, ok = flags[cmdFlag("Int")]
	require.True(t, ok)
	require.Equal(t, 2, len(f.Annotations))
	require.Equal(t, "one", f.Annotations[0])
	require.Equal(t, "two", f.Annotations[1])

	args, err := GetCmdArgSet(c)
	require.NoError(t, err)
	require.Equal(t, 1, len(args.Flags))
	f = args.Flags[0]
	require.Equal(t, cmdFlag("Arg"), f.Name)
	require.Equal(t, 1, len(f.Annotations))
	require.Equal(t, "ok", f.Annotations[0])
}

// TestBindPtrBoolIntString test that binding to pointer values
func TestBindPtrBoolIntString(t *testing.T) {

	// start with un-initialized pointers
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestPtrBoolIntString{}
	err := Bind(c, sts)
	// binding to nil is supported
	require.NoError(t, err)
	pfbool := assertFlag(t, c, "Is")
	pfint := assertFlag(t, c, "Int")
	pfstr := assertFlag(t, c, "Str")
	pfint64 := assertFlag(t, c, "Int64")
	require.Nil(t, sts.Is)
	require.Nil(t, sts.Int)
	require.Nil(t, sts.Int64)
	require.Nil(t, sts.Str)

	err = pfbool.Value.Set("3")
	require.Error(t, err)
	require.Nil(t, sts.Is)
	err = pfbool.Value.Set("true")
	require.NoError(t, err)
	require.True(t, *sts.Is)

	err = pfint.Value.Set("bla")
	require.Error(t, err)
	require.Nil(t, sts.Int)
	err = pfint.Value.Set("3")
	require.NoError(t, err)
	require.Equal(t, 3, *sts.Int)

	err = pfint64.Value.Set("bla")
	require.Error(t, err)
	require.Nil(t, sts.Int64)
	err = pfint64.Value.Set("64")
	require.NoError(t, err)
	require.Equal(t, int64(64), *sts.Int64)

	err = pfstr.Value.Set("")
	require.NoError(t, err)
	require.Equal(t, "", *sts.Str)
	err = pfstr.Value.Set("hello")
	require.NoError(t, err)
	require.Equal(t, "hello", *sts.Str)

	// start with initialized pointers
	c = &cobra.Command{
		Use: "dontUse",
	}
	sts = &TestPtrBoolIntString{
		Is:    new(bool),
		Int:   new(int),
		Int64: new(int64),
		Str:   new(string),
	}
	err = Bind(c, sts)
	require.NoError(t, err)

	require.NoError(t, err)
	pfbool = assertFlag(t, c, "Is")
	pfint = assertFlag(t, c, "Int")
	pfstr = assertFlag(t, c, "Str")
	pfint64 = assertFlag(t, c, "Int64")

	require.False(t, *sts.Is)
	err = pfbool.Value.Set("true")
	require.NoError(t, err)
	require.True(t, *sts.Is)

	require.Equal(t, 0, *sts.Int)
	err = pfint.Value.Set("3")
	require.NoError(t, err)
	require.Equal(t, 3, *sts.Int)

	require.Equal(t, int64(0), *sts.Int64)
	err = pfint64.Value.Set("64")
	require.NoError(t, err)
	require.Equal(t, int64(64), *sts.Int64)

	require.Equal(t, "", *sts.Str)
	err = pfstr.Value.Set("hello")
	require.NoError(t, err)
	require.Equal(t, "hello", *sts.Str)
}

type TestPtrBoolStruct struct {
	Is     *bool `cmd:"flag,is"`
	Ignore string
}

// TestBindPtrBool test that binding to bool pointer values works
// It also test the cobra command line parser behavior.
func TestBindPtrBool(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestPtrBoolStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "is")

	require.Nil(t, sts.Is)

	err = pf.Value.Set("true")
	require.NoError(t, err)
	require.NotNil(t, sts.Is)
	require.True(t, *sts.Is)

	err = pf.Value.Set("false")
	require.NoError(t, err)
	require.False(t, *sts.Is)

	err = pf.Value.Set("xfalse")
	require.Error(t, err)

	// parse
	c = &cobra.Command{
		Use: "dontUse",
	}
	sts = &TestPtrBoolStruct{}
	err = Bind(c, sts)
	require.NoError(t, err)
	pf = assertFlag(t, c, "is")
	fs := pflag.FlagSet{}
	fs.AddFlag(pf)
	err = fs.Parse([]string{"--is"})
	require.NoError(t, err)
	require.NotNil(t, sts.Is)
	require.True(t, *sts.Is)

	err = fs.Parse([]string{"--is=false"})
	require.NoError(t, err)
	require.NotNil(t, sts.Is)
	require.False(t, *sts.Is)
}

type TestBoolStruct struct {
	Is     bool `cmd:"flag"`
	Ignore string
}

func TestBindBool(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestBoolStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "Is")

	require.False(t, sts.Is)
	err = pf.Value.Set("true")
	require.NoError(t, err)
	require.True(t, sts.Is)
}

type TestStringStruct struct {
	Name   string `cmd:"flag"`
	Ignore string
}

func TestBindString(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestStringStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "Name")

	require.Equal(t, "", sts.Name)
	err = pf.Value.Set("john")
	require.NoError(t, err)
	require.Equal(t, "john", sts.Name)
}

type TestUintStruct struct {
	Uint   uint   `cmd:"flag"`
	Uint8  uint8  `cmd:"flag"`
	Uint16 uint16 `cmd:"flag"`
	Uint32 uint32 `cmd:"flag"`
	Uint64 uint64 `cmd:"flag"`
	Ignore string
}

func TestBindUint(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestUintStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "Uint")
	require.Equal(t, uint(0), sts.Uint)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, uint(2), sts.Uint)

	pf = assertFlag(t, c, "Uint8")
	require.Equal(t, uint8(0), sts.Uint8)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, uint8(2), sts.Uint8)

	pf = assertFlag(t, c, "Uint16")
	require.Equal(t, uint16(0), sts.Uint16)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, uint16(2), sts.Uint16)

	pf = assertFlag(t, c, "Uint32")
	require.Equal(t, uint32(0), sts.Uint32)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, uint32(2), sts.Uint32)

	pf = assertFlag(t, c, "Uint64")
	require.Equal(t, uint64(0), sts.Uint64)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, uint64(2), sts.Uint64)
}

type TestIntStruct struct {
	Int    int   `cmd:"flag"`
	Int8   int8  `cmd:"flag"`
	Int16  int16 `cmd:"flag"`
	Int32  int32 `cmd:"flag"`
	Int64  int64 `cmd:"flag"`
	Ignore string
}

func TestBindInt(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestIntStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "Int")
	require.Equal(t, 0, sts.Int)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, 2, sts.Int)

	pf = assertFlag(t, c, "Int8")
	require.Equal(t, int8(0), sts.Int8)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, int8(2), sts.Int8)

	pf = assertFlag(t, c, "Int16")
	require.Equal(t, int16(0), sts.Int16)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, int16(2), sts.Int16)

	pf = assertFlag(t, c, "Int32")
	require.Equal(t, int32(0), sts.Int32)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, int32(2), sts.Int32)

	pf = assertFlag(t, c, "Int64")
	require.Equal(t, int64(0), sts.Int64)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, int64(2), sts.Int64)
}

type TestFloatStruct struct {
	Float32 float32 `cmd:"flag"`
	Float64 float64 `cmd:"flag"`
	Ignore  string
}

func TestBindFloat(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestFloatStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "Float32")
	require.Equal(t, float32(0), sts.Float32)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, float32(2), sts.Float32)

	pf = assertFlag(t, c, "Float64")
	require.Equal(t, float64(0), sts.Float64)
	err = pf.Value.Set("2")
	require.NoError(t, err)
	require.Equal(t, float64(2), sts.Float64)
}

type TestIPAndDurationStruct struct {
	IP       net.IP        `cmd:"flag"`
	Duration time.Duration `cmd:"flag"`
	Ignore   string
}

func TestBindIP(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestIPAndDurationStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "IP")
	require.Equal(t, 0, len(sts.IP))
	err = pf.Value.Set("192.0.2.1")
	require.NoError(t, err)
	expectedIP := net.ParseIP("192.0.2.1")
	require.Equal(t, expectedIP, sts.IP)

	pf = assertFlag(t, c, "Duration")
	require.Equal(t, time.Duration(0), sts.Duration)
	err = pf.Value.Set("300ms")
	require.NoError(t, err)
	expectedD, err := time.ParseDuration("300ms")
	require.NoError(t, err)
	require.Equal(t, expectedD, sts.Duration)
}

type TestArgNoOrderStruct struct {
	ID     string `cmd:"arg"`
	Token  string `cmd:"arg"`
	Hash   string `cmd:"arg"`
	Ignore string
}

func TestArgNoOrder(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestArgNoOrderStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)
	v, err := SetArgs(c, []string{"one", "two", "three"})
	require.NoError(t, err)
	aos, ok := v.(*TestArgNoOrderStruct)
	require.True(t, ok)
	require.Equal(t, "one", aos.ID)
	require.Equal(t, "two", aos.Token)
	require.Equal(t, "three", aos.Hash)
}

type TestArgOrderStruct struct {
	ID     string `cmd:"arg,,,1"`
	Token  string `cmd:"arg,,,0"`
	Hash   string `cmd:"arg,,,2"`
	Ignore string
}

func TestArgOrder(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestArgOrderStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)
	v, err := SetArgs(c, []string{"one", "two", "three"})
	require.NoError(t, err)
	aos, ok := v.(*TestArgOrderStruct)
	require.True(t, ok)
	require.Equal(t, "two", aos.ID)
	require.Equal(t, "one", aos.Token)
	require.Equal(t, "three", aos.Hash)
}

type TestArgBadOrderStruct struct {
	ID     string `cmd:"arg"`
	Token  string `cmd:"arg,,,0"`
	Hash   string `cmd:"arg,,,2"`
	Ignore string
}

func TestArgBadOrder(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestArgBadOrderStruct{}
	err := Bind(c, sts)
	require.Error(t, err)
}

type TestVariadicStringStruct struct {
	ID     string   `cmd:"arg,,,1"`
	Token  string   `cmd:"arg,,,0"`
	Hashes []string `cmd:"arg,,,2"`
	Ignore string
}

func TestVariadicString(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestVariadicStringStruct{
		Hashes: []string{"hip", "hop"},
	}
	err := Bind(c, sts)
	require.NoError(t, err)
	v, err := SetArgs(c, []string{"one", "two", "three,four,five,six"})
	require.NoError(t, err)
	aos, ok := v.(*TestVariadicStringStruct)
	require.True(t, ok)
	require.Equal(t, "two", aos.ID)
	require.Equal(t, "one", aos.Token)
	require.Equal(t, []string{"three", "four", "five", "six"}, aos.Hashes)

	// reset hashes since pflag append values if they were previously changed !!
	// see: pflag/string_slice.go
	sts.Hashes = nil
	v, err = SetArgs(c, []string{"bone", "btwo", "bthree", "bfour", "bfive", "bsix"})
	require.NoError(t, err)
	aos, ok = v.(*TestVariadicStringStruct)
	require.True(t, ok)
	require.Equal(t, "btwo", aos.ID)
	require.Equal(t, "bone", aos.Token)
	require.Equal(t, []string{"bthree", "bfour", "bfive", "bsix"}, aos.Hashes)
}

type TestVariadicIntStruct struct {
	ID     string `cmd:"arg,,,1"`
	Token  string `cmd:"arg,,,0"`
	Ints   []int  `cmd:"arg,,,2"`
	Ignore string
}

func TestVariadicInt(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestVariadicIntStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)
	v, err := SetArgs(c, []string{"one", "two", "3,4,5,6"})
	require.NoError(t, err)
	require.Equal(t, &TestVariadicIntStruct{
		ID:    "two",
		Token: "one",
		Ints:  []int{3, 4, 5, 6},
	}, v)

	// reset to workaround append - see: pflag/int_slice.go
	sts.Ints = nil
	v, err = SetArgs(c, []string{"bone", "btwo", "13", "14", "15", "16"})
	require.NoError(t, err)
	require.Equal(t, &TestVariadicIntStruct{
		ID:    "btwo",
		Token: "bone",
		Ints:  []int{13, 14, 15, 16},
	}, v)
}

type TestCommonSliceStruct struct {
	Hashes []string `cmd:"flag"`
	Ints   []int    `cmd:"flag"`
	Bools  []bool   `cmd:"flag"`
	Ignore string
}

func TestBindCommonSlice(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestCommonSliceStruct{}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "Hashes")
	require.Empty(t, sts.Hashes)
	err = pf.Value.Set("1234,5678")
	require.NoError(t, err)
	hashes := []string{"1234", "5678"}
	require.Equal(t, hashes, sts.Hashes)
	/* string slices are not trimmed
	err = pf.Value.Set("1234 ,  5678")
	...
	*/

	pf = assertFlag(t, c, "Ints")
	require.Empty(t, sts.Ints)
	err = pf.Value.Set("1234,5678")
	require.NoError(t, err)
	ints := []int{1234, 5678}
	require.Equal(t, ints, sts.Ints)
	/* int slices are not trimmed
	err = pf.Value.Set("1234 ,  5678")
	require.NoError(t, err)
	require.Equal(t, ints, sts.Ints)
	*/

	pf = assertFlag(t, c, "Bools")
	require.Empty(t, sts.Bools)
	err = pf.Value.Set("true,false")
	require.NoError(t, err)
	bools := []bool{true, false}
	require.Equal(t, bools, sts.Bools)
	sts.Bools = nil
	err = pf.Value.Set("false  ,  true  ")
	require.NoError(t, err)
	bools = []bool{false, true}
	require.Equal(t, bools, sts.Bools)
}

// --- pointer flag value ---

// workersConfig implements pflag.Value
type workersConfig struct {
	QueueSize int
	Workers   int
}

func newWorkersConfig() *workersConfig {
	return &workersConfig{
		QueueSize: 50,
		Workers:   5,
	}
}

func (u *workersConfig) String() string {
	bb, err := json.Marshal(u)
	if err != nil {
		return err.Error()
	}
	return string(bb)
}

func (u *workersConfig) Set(s string) error {
	return json.Unmarshal([]byte(s), u)
}

func (u *workersConfig) Type() string {
	return "workers"
}

type TestFlagValueStruct struct {
	Workers *workersConfig `cmd:"flag"`
}

func TestBindFlagValue(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestFlagValueStruct{
		Workers: &workersConfig{},
	}
	err := Bind(c, sts)
	require.NoError(t, err)

	pf := assertFlag(t, c, "Workers")
	require.Equal(t, &workersConfig{}, sts.Workers)
	err = pf.Value.Set(`{"QueueSize":50,"Workers":5}`)
	require.NoError(t, err)
	wc := newWorkersConfig()
	require.Equal(t, wc, sts.Workers)
}

// --- non pointer flag value (not supported for now) ---
type TestFlagValueStruct2 struct {
	Workers workersConfig2 `cmd:"flag"`
}

/* ==
func TestFlagValueNonPointer(t *testing.T) {
	sts2 := &TestFlagValueStruct2{}
	sts2.Workers.QueueSize = 2
	v := reflect.ValueOf(sts2).Elem()

	fmt.Println("type", v.Type(), "kind", v.Kind())
	fv := v.Field(0)
	fmt.Println("type", fv.Type(), "kind", fv.Kind())

	iface, ok := fv.Interface().(flag.Value)
	// same with that:
	//iface, ok := fv.Addr().Interface().(flag.Value)
	fmt.Println("ok", ok, iface, fv.CanAddr())

	ptr := reflect.New(fv.Type())
	iface, ok = ptr.Elem().Interface().(flag.Value)
	fmt.Println("ok", ok, iface)

	err := iface.Set(`{"QueueSize":50,"Workers":50} `)
	require.NoError(t, err)
	fmt.Println(iface)

	fv.Set(reflect.ValueOf(iface))
	fmt.Println(sts2.Workers)
}

func newWorkersConfig2() workersConfig2 {
	return workersConfig2{
		QueueSize: 50,
		Workers:   5,
	}
}

*/

type workersConfig2 struct {
	QueueSize int
	Workers   int
}

func (u workersConfig2) empty() bool {
	return u.Workers <= 0 && u.QueueSize <= 0
}

func (u workersConfig2) String() string {
	bb, err := json.Marshal(u)
	if err != nil {
		return err.Error()
	}
	return string(bb)
}

func (u workersConfig2) Set(s string) error {
	return json.Unmarshal([]byte(s), &u)
}

func (u workersConfig2) Type() string {
	return "workers"
}

func TestBindFlagValueNonPointer(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts2 := &TestFlagValueStruct2{}
	err := Bind(c, sts2)
	//
	// we don't support non pointer struct implementing flag.Value for now
	//
	require.NoError(t, err)
	/* maybe later */
	//pf := assertFlag(t, c, "Workers")

	flags, err := GetCmdFlagSet(c)
	require.NoError(t, err)
	_, ok := flags[cmdFlag("Workers")]
	require.False(t, ok) // maybe later
	/**
	require.Equal(t, workersConfig2{}, sts2.Workers)
	err = pf.Value.Set(`{"QueueSize":50,"Workers":5}`)
	require.NoError(t, err)
	wc := newWorkersConfig2()
	require.Equal(t, wc, sts2.Workers)
	*/
}

// --- anonymous inner struct ---

type worker struct {
	Size int    `cmd:"flag"`
	Name string `cmd:"flag"`
}

type TestFlagAnonStruct struct {
	*worker
	Activity string `cmd:"flag"`
}

// NOTE: the anonymous field needs to be initialized !
// 		 otherwise no binding occurs
func emptyTestFlagAnonStruct() *TestFlagAnonStruct {
	return &TestFlagAnonStruct{worker: &worker{}}
}

func TestBindAnonymousInnerStruct(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := emptyTestFlagAnonStruct()
	err := Bind(c, sts)
	require.NoError(t, err)
	require.Equal(t, emptyTestFlagAnonStruct(), sts)

	pfSize := assertFlag(t, c, "Size")
	pfName := assertFlag(t, c, "Name")
	pfActv := assertFlag(t, c, "Activity")

	err = pfSize.Value.Set("50")
	require.NoError(t, err)
	err = pfName.Value.Set("john")
	require.NoError(t, err)
	err = pfActv.Value.Set("student")
	require.NoError(t, err)

	exp := &TestFlagAnonStruct{
		worker: &worker{
			Size: 50,
			Name: "john",
		},
		Activity: "student",
	}
	require.Equal(t, exp, sts)

	// test with un-initialized inner struct
	c = &cobra.Command{
		Use: "dontUse",
	}
	sts = &TestFlagAnonStruct{}
	err = Bind(c, sts)
	require.Error(t, err)
	require.True(t, errors.IsKind(errors.K.Invalid, err))
}

type TestFlagAnonStructValue struct {
	worker
	Activity string `cmd:"flag"`
}

func TestBindAnonymousInnerStructValue(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestFlagAnonStructValue{}
	err := Bind(c, sts)
	require.NoError(t, err)
	require.Equal(t, &TestFlagAnonStructValue{}, sts)

	pfSize := assertFlag(t, c, "Size")
	pfName := assertFlag(t, c, "Name")
	pfActv := assertFlag(t, c, "Activity")

	err = pfSize.Value.Set("50")
	require.NoError(t, err)
	err = pfName.Value.Set("john")
	require.NoError(t, err)
	err = pfActv.Value.Set("student")
	require.NoError(t, err)

	exp := &TestFlagAnonStructValue{
		worker: worker{
			Size: 50,
			Name: "john",
		},
		Activity: "student",
	}
	require.Equal(t, exp, sts)
}

type InnerStruct struct {
	Name string `cmd:"flag"`
	Age  int    `cmd:"flag"`
}

type TestPtrStruct struct {
	Inner    *InnerStruct
	Activity string `cmd:"flag"`
	Ignore   string
}

func TestBindPtrStruct(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestPtrStruct{}
	err := Bind(c, sts)
	// binding to nil not supported
	require.Error(t, err)

	c = &cobra.Command{
		Use: "dontUse",
	}
	sts = &TestPtrStruct{
		Inner: &InnerStruct{},
	}
	err = Bind(c, sts)
	require.NoError(t, err)

	require.NoError(t, err)
	pfName := assertFlag(t, c, "Name")
	pfAge := assertFlag(t, c, "Age")
	pfActivity := assertFlag(t, c, "Activity")

	err = pfName.Value.Set("john")
	require.NoError(t, err)
	err = pfAge.Value.Set("50")
	require.NoError(t, err)
	err = pfActivity.Value.Set("student")
	require.NoError(t, err)

	exp := &TestPtrStruct{
		Activity: "student",
		Inner: &InnerStruct{
			Name: "john",
			Age:  50,
		},
	}
	require.Equal(t, exp, sts)
}

type InnerStructWithPtr struct {
	Name *string `cmd:"flag"`
	Age  *int    `cmd:"flag"`
}

type TestPtrStructWithInnerPtr struct {
	Inner    *InnerStructWithPtr
	Activity string `cmd:"flag"`
	Ignore   string
}

func TestBindPtrStructWithInnerPtr(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestPtrStructWithInnerPtr{}
	err := Bind(c, sts)
	// binding to nil not supported
	require.Error(t, err)

	c = &cobra.Command{
		Use: "dontUse",
	}
	sts = &TestPtrStructWithInnerPtr{
		Inner: &InnerStructWithPtr{
			Name: new(string),
			Age:  new(int),
		},
	}
	err = Bind(c, sts)
	require.NoError(t, err)

	require.NoError(t, err)
	pfName := assertFlag(t, c, "Name")
	pfAge := assertFlag(t, c, "Age")
	pfActivity := assertFlag(t, c, "Activity")

	err = pfName.Value.Set("john")
	require.NoError(t, err)
	err = pfAge.Value.Set("50")
	require.NoError(t, err)
	err = pfActivity.Value.Set("student")
	require.NoError(t, err)

	name := new(string)
	*name = "john"
	age := new(int)
	*age = 50

	exp := &TestPtrStructWithInnerPtr{
		Activity: "student",
		Inner: &InnerStructWithPtr{
			Name: name,
			Age:  age,
		},
	}
	require.Equal(t, exp, sts)
}
