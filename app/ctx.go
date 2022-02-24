package app

const (
	CtxCmd           = "cmd"
	CtxResult        = "result"
	CtxAddResultFn   = "add-result-fn"
	CtxPrintResultFn = "print-result-fn"
	CmdValidate      = "$cmd-validate"
)

// ----- CmdCtx -----

// CmdCtx implements a simple map to get/set objects by name
type CmdCtx struct {
	kv map[string]interface{}
}

func NewCmdCtx() *CmdCtx {
	return &CmdCtx{
		kv: make(map[string]interface{}),
	}
}

func (c *CmdCtx) Get(k string) (interface{}, bool) {
	if c == nil {
		return nil, false
	}
	v, ok := c.kv[k]
	return v, ok
}

func (c *CmdCtx) Set(k string, v interface{}) {
	c.kv[k] = v
}
