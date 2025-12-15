package js

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"

	"github.com/docker/cagent/pkg/environment"
)

// escapeForTemplateLiteral escapes characters that have special meaning in
// JavaScript template literals
func escapeForTemplateLiteral(s string) string {
	// Escape backslashes first (must be done before backticks to avoid double-escaping)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	// Then escape backticks so they don't terminate the template literal
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

type jsEnv func(string) goja.Value

func (e jsEnv) Get(k string) goja.Value     { return e(k) }
func (e jsEnv) Set(string, goja.Value) bool { return false }
func (e jsEnv) Has(string) bool             { return true }
func (e jsEnv) Delete(string) bool          { return true }
func (e jsEnv) Keys() []string              { return nil }

type Expander struct {
	env environment.Provider

	vm   *goja.Runtime
	lock sync.Once
}

func NewJsExpander(env environment.Provider) *Expander {
	return &Expander{
		env: env,
	}
}

func (exp *Expander) jsRuntime(ctx context.Context) *goja.Runtime {
	exp.lock.Do(func() {
		vm := goja.New()
		_ = vm.Set("env", vm.NewDynamicObject(jsEnv(func(k string) goja.Value {
			return vm.ToValue(exp.env.Get(ctx, k))
		})))

		exp.vm = vm
	})

	return exp.vm
}

func (exp *Expander) ExpandMap(ctx context.Context, kv map[string]string) map[string]string {
	expanded := map[string]string{}

	vm := exp.jsRuntime(ctx)

	for k, v := range kv {
		result, err := vm.RunString("`" + escapeForTemplateLiteral(v) + "`")
		if err != nil {
			expanded[k] = v
			continue
		}

		expanded[k] = fmt.Sprintf("%v", result.Export())
	}

	return expanded
}

func (exp *Expander) Expand(ctx context.Context, text string) string {
	vm := exp.jsRuntime(ctx)

	result, err := vm.RunString("`" + escapeForTemplateLiteral(text) + "`")
	if err != nil {
		return text
	}

	return fmt.Sprintf("%v", result.Export())
}

func ExpandString(_ context.Context, str string, values map[string]string) (string, error) {
	vm := goja.New()

	for k, v := range values {
		_ = vm.Set(k, v)
	}

	expanded, err := vm.RunString("`" + escapeForTemplateLiteral(str) + "`")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", expanded.Export()), nil
}
