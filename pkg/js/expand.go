package js

import (
	"context"
	"strings"

	"github.com/dop251/goja"

	"github.com/docker/cagent/pkg/config/types"
	"github.com/docker/cagent/pkg/environment"
)

// newVM creates a new Goja JavaScript runtime.
var newVM = goja.New

// Expander expands JavaScript template literals in strings using environment variables.
type Expander struct {
	env environment.Provider
}

// NewJsExpander creates a new Expander with the given environment provider.
func NewJsExpander(env environment.Provider) *Expander {
	return &Expander{
		env: env,
	}
}

// dynamicLookup implements goja.DynamicObject for lazy key-value access.
type dynamicLookup struct {
	vm     *goja.Runtime
	lookup func(string) string
}

func (d *dynamicLookup) Get(k string) goja.Value   { return d.vm.ToValue(d.lookup(k)) }
func (*dynamicLookup) Set(string, goja.Value) bool { return false }
func (*dynamicLookup) Has(string) bool             { return true }
func (*dynamicLookup) Delete(string) bool          { return true }
func (*dynamicLookup) Keys() []string              { return nil }

// newEnvVM creates a new JS runtime with the 'env' dynamic object pre-bound
// to the Expander's environment provider.
func (exp *Expander) newEnvVM(ctx context.Context) *goja.Runtime {
	vm := newVM()
	_ = vm.Set("env", vm.NewDynamicObject(&dynamicLookup{
		vm:     vm,
		lookup: func(k string) string { v, _ := exp.env.Get(ctx, k); return v },
	}))
	return vm
}

// Expand expands JavaScript template literals using the provided values map.
// The values are bound as top-level variables in the JS runtime alongside the
// env object from the Expander's environment provider.
func (exp *Expander) Expand(ctx context.Context, text string, values map[string]string) string {
	if !strings.Contains(text, "${") {
		return text
	}

	vm := exp.newEnvVM(ctx)
	for k, v := range values {
		_ = vm.Set(k, v)
	}

	return runExpansion(vm, text)
}

// ExpandMap expands JavaScript template literals in all values of the given map.
func (exp *Expander) ExpandMap(ctx context.Context, kv map[string]string) map[string]string {
	if kv == nil {
		return nil
	}

	vm := exp.newEnvVM(ctx)

	expanded := make(map[string]string, len(kv))
	for k, v := range kv {
		expanded[k] = runExpansion(vm, v)
	}
	return expanded
}

// ExpandCommands expands JavaScript template literals in all command fields.
func (exp *Expander) ExpandCommands(ctx context.Context, cmds types.Commands) types.Commands {
	if cmds == nil {
		return nil
	}

	vm := exp.newEnvVM(ctx)

	expanded := make(types.Commands, len(cmds))
	for k, cmd := range cmds {
		expanded[k] = types.Command{
			Description: runExpansion(vm, cmd.Description),
			Instruction: runExpansion(vm, cmd.Instruction),
		}
	}
	return expanded
}

// ExpandMapFunc expands JavaScript template literals in map values.
// It binds a dynamic object with the given name to the JS runtime,
// using lookup to resolve property accesses. Each value is optionally
// preprocessed with preprocess before expansion (pass nil to skip).
func ExpandMapFunc(values map[string]string, objName string, lookup, preprocess func(string) string) map[string]string {
	vm := newVM()
	_ = vm.Set(objName, vm.NewDynamicObject(&dynamicLookup{
		vm:     vm,
		lookup: lookup,
	}))

	resolved := make(map[string]string, len(values))
	for k, v := range values {
		if preprocess != nil {
			v = preprocess(v)
		}
		resolved[k] = runExpansion(vm, v)
	}
	return resolved
}

// runExpansion executes the template string using the provided Goja runtime.
func runExpansion(vm *goja.Runtime, text string) string {
	// Escape backslashes first, then backticks
	escaped := strings.ReplaceAll(text, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	script := "`" + escaped + "`"

	v, err := vm.RunString(script)
	if err != nil {
		return text
	}

	if v == nil || v.Export() == nil {
		return ""
	}

	return v.String()
}
