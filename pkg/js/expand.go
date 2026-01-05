package js

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"

	"github.com/docker/cagent/pkg/config/types"
	"github.com/docker/cagent/pkg/environment"
)

// Expander expands JavaScript template literals in strings using environment variables.
type Expander struct {
	env environment.Provider
}

// NewJsExpander creates a new Expander with the given environment provider.
func NewJsExpander(env environment.Provider) *Expander {
	return &Expander{env: env}
}

// dynamicEnv implements goja.DynamicObject for lazy environment variable access.
type dynamicEnv func(string) goja.Value

func (e dynamicEnv) Get(k string) goja.Value   { return e(k) }
func (dynamicEnv) Set(string, goja.Value) bool { return false }
func (dynamicEnv) Has(string) bool             { return true }
func (dynamicEnv) Delete(string) bool          { return true }
func (dynamicEnv) Keys() []string              { return nil }

// Expand expands JavaScript template literals in the given text.
func (exp *Expander) Expand(ctx context.Context, text string) string {
	if !strings.Contains(text, "${") {
		return text
	}

	vm := goja.New()
	_ = vm.Set("env", vm.NewDynamicObject(dynamicEnv(func(k string) goja.Value {
		v, _ := exp.env.Get(ctx, k)
		return vm.ToValue(v)
	})))

	return runExpansion(vm, text)
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
	return fmt.Sprintf("%v", v.Export())
}

// ExpandMap expands JavaScript template literals in all values of the given map.
func (exp *Expander) ExpandMap(ctx context.Context, kv map[string]string) map[string]string {
	expanded := make(map[string]string, len(kv))
	for k, v := range kv {
		expanded[k] = exp.Expand(ctx, v)
	}
	return expanded
}

// ExpandCommands expands JavaScript template literals in all command fields.
func (exp *Expander) ExpandCommands(ctx context.Context, cmds types.Commands) types.Commands {
	if cmds == nil {
		return nil
	}

	expanded := make(types.Commands, len(cmds))
	for k, cmd := range cmds {
		expanded[k] = types.Command{
			Description: exp.Expand(ctx, cmd.Description),
			Instruction: exp.Expand(ctx, cmd.Instruction),
		}
	}
	return expanded
}

// ExpandString expands JavaScript template literals using the provided values map.
func ExpandString(_ context.Context, str string, values map[string]string) (string, error) {
	if !strings.Contains(str, "${") {
		return str, nil
	}

	vm := goja.New()
	for k, v := range values {
		_ = vm.Set(k, v)
	}

	return runExpansion(vm, str), nil
}
