package v8_test

import (
	"fmt"

	"github.com/augustoroman/v8"
)

func Example() {
	// Easy-peasy to create a new VM:
	ctx := v8.NewIsolate().NewContext()

	// You can load your js from a file, create it dynamically, whatever.
	ctx.Eval(`
            // This is javascript code!
            add = (a,b)=>{ return a + b }; // whoa, ES6 arrow functions.
        `, "add.js") // <-- supply filenames for stack traces

	// State accumulates in a context.  Add still exists.
	// The last statements' value is returned to Go.
	res, _ := ctx.Eval(`add(3,4)`, "compute.js") // don't ignore errors!
	fmt.Println("add(3,4) =", res.String())      // I hope it's 7.

	// You can also bind Go functions to javascript:
	my_count_function := func(in v8.CallbackArgs) (*v8.Value, error) {
		return in.Context.Create(len(in.Args)) // ctx.Create is great for mapping Go -> JS.
	}
	cnt := ctx.Bind("count", my_count_function)
	ctx.Global().Set("count_args", cnt)

	res, _ = ctx.Eval(`
            // Now we can call that function in JS
            count_args(1,2,3,4,5)
        `, "compute2.js")

	fmt.Println("count_args(1,2,3,4,5) =", res.String())

	_, err := ctx.Eval(`
            // Sometimes there's a mistake in your js code:
            functin broken(a,b) { return a+b; }
        `, "ooops.js")
	fmt.Println("Err:", err) // <-- get nice error messages

	// output:
	// add(3,4) = 7
	// count_args(1,2,3,4,5) = 5
	// Err: Uncaught exception: SyntaxError: Unexpected identifier
	// at ooops.js:3:20
	//               functin broken(a,b) { return a+b; }
	//                       ^^^^^^
	// Stack trace: SyntaxError: Unexpected identifier
}

func ExampleContext_Create_basic() {
	ctx := v8.NewIsolate().NewContext()

	type Info struct{ Name, Email string }

	val, _ := ctx.Create(map[string]interface{}{
		"num":    3.7,
		"str":    "simple string",
		"bool":   true,
		"struct": Info{"foo", "bar"},
		"list":   []int{1, 2, 3},
	})

	// val is now a *v8.Value that is associated with ctx but not yet accessible
	// from the javascript scope.

	_ = ctx.Global().Set("created_value", val)

	res, _ := ctx.Eval(`
            created_value.struct.Name = 'John';
            JSON.stringify(created_value.struct)
        `, `test.js`)
	fmt.Println(res)

	// output:
	// {"Name":"John","Email":"bar"}
}

func ExampleContext_Create_callbacks() {
	ctx := v8.NewIsolate().NewContext()

	// A typical use of Create is to return values from callbacks:
	var nextId int
	getNextIdCallback := func(in v8.CallbackArgs) (*v8.Value, error) {
		nextId++
		return ctx.Create(nextId) // Return the created corresponding v8.Value or an error.
	}

	// Because Create will use reflection to map a Go value to a JS object, it
	// can also be used to easily bind a complex object into the JS VM.
	resetIdsCallback := func(in v8.CallbackArgs) (*v8.Value, error) {
		nextId = 0
		return ctx.Create(0)
	}
	myIdAPI, _ := ctx.Create(map[string]interface{}{
		"next":  getNextIdCallback,
		"reset": resetIdsCallback,
		// Can also include other stuff:
		"my_api_version": "v1.2",
	})

	// now let's use those two callbacks and the api value:
	_ = ctx.Global().Set("ids", myIdAPI)
	var res *v8.Value
	res, _ = ctx.Eval(`ids.my_api_version`, `test.js`)
	fmt.Println(`ids.my_api_version =`, res)
	res, _ = ctx.Eval(`ids.next()`, `test.js`)
	fmt.Println(`ids.next() =`, res)
	res, _ = ctx.Eval(`ids.next()`, `test.js`)
	fmt.Println(`ids.next() =`, res)
	res, _ = ctx.Eval(`ids.reset(); ids.next()`, `test.js`)
	fmt.Println(`ids.reset()`)
	fmt.Println(`ids.next() =`, res)

	// output:
	// ids.my_api_version = v1.2
	// ids.next() = 1
	// ids.next() = 2
	// ids.reset()
	// ids.next() = 1
}

func ExampleSnapshot() {
	snapshot := v8.CreateSnapshot(`
        // Concantenate all the scripts you want at startup, e.g. lodash, etc.
        _ = { map: function() { /* ... */ }, etc: "etc, etc..." };
        // Setup my per-context global state:
        myGlobalState = {
            init: function() { this.initialized = true; },
            foo: 3,
        };
        // Run some functions:
        myGlobalState.init();
    `)
	iso := v8.NewIsolateWithSnapshot(snapshot)

	// Create a context with the state from the snapshot:
	ctx1 := iso.NewContext()
	fmt.Println("Context 1:")
	val, _ := ctx1.Eval("myGlobalState.foo = 37; myGlobalState.initialized", "")
	fmt.Println("myGlobalState.initialized:", val)
	val, _ = ctx1.Eval("myGlobalState.foo", "")
	fmt.Println("myGlobalState.foo:", val)

	// In the second context, the global state is reset to the state at the
	// snapshot:
	ctx2 := iso.NewContext()
	fmt.Println("Context 2:")
	val, _ = ctx2.Eval("myGlobalState.foo", "")
	fmt.Println("myGlobalState.foo:", val)

	// Output:
	// Context 1:
	// myGlobalState.initialized: true
	// myGlobalState.foo: 37
	// Context 2:
	// myGlobalState.foo: 3
}
