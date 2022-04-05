<!--
tags: [ "ruby", "go", "ffi" ]
date_created: "2022-04-05T22:42:00"
image: /static/images/gopher-portal.png
-->

# A portal between Ruby and Go (using FFI)

Thinking about migrating some huge REST APIs from Ruby to Go, I researched how to replace the Ruby code progressively with Go code.

I ended up with a possible FFI-based prototype. The solution is to have the Ruby code load a Go library and wrap the Go functions in Ruby classes and methods.

## The Ruby side

The Ruby side is pretty straightforward. We must load the Go library and import the Go functions in Ruby classes and methods.

<pre class="highlight">
require 'ffi'

module Portal
  extend FFI::Library

  ffi_lib './libexample.so'

  class Example < FFI::Struct
    # This must be completely in sync with the C struct defined in Go code.
    layout :id, :int, :prefix, :pointer
  
    def initialize(prefix, id)
      self[:prefix] = FFI::MemoryPointer.from_string(prefix)
      self[:id] = id
    end

    # This feels convoluted, but it hides the fact that our function is loaded
    # outside of the "struct mirror" class.
    def greet
      Portal.greet(self)
    end
  end

  attach_function 'greet', [Example.by_value], :void
end

ex = Portal::Example.new('C', 137)
ex.greet
</pre>

## The Go side

The Go side is a bit more complex. We need to define a C-compatible struct and export the functions we want to use from Ruby.

The cool thing is that we can define the functions with the struct as the receiver.

<pre class="highlight">
package main

/*
struct example {
	int ID;
	char *Prefix;
};
*/
import "C"
import "fmt"

// This declaration is just an alias to the C struct.
type Example C.struct_example

//export greet
func (e Example) greet() {
	fmt.Printf("Hello from %s-%d\n", C.GoString(e.Prefix), e.ID)
}

func main() {}
</pre>

## The build

The build is also straightforward. We need to:

1. Compile the Go code to a shared library. This command will generate the CGO bindings.
2. Run the Ruby code.

<pre class="highlight">
go build -buildmode=c-shared -o libexample.so example.go
ruby portal.rb
</pre>

## The result

If we run the code, we get the following output:

<pre class="highlight">
Hello from C-137
</pre>

## The benefits

The main benefit of this approach is that we can progressively migrate the Ruby code to Go. We can start by replacing part by part, until we are ready to switch to a pure Go service.

It's not only possible to replace existing Ruby code. Now we can reuse RSpec/Minitest specs to test the new Go code. Instead of going for a full rewrite with no tests, we can start by testing the new code with the existing ones.

## The (possible) drawbacks

Performance-wise this approach is not ideal. We are adding a layer of indirection. This probably adds some overhead to the calls. This is something we need to measure if applied to high-traffic services.

<img src="/static/images/gopher-portal.png" alt="Image generated with DALL-E and the prompt 'A digital illustration of a blue gopher staring to a wobbly shimmering Rick and Morty style portal. Eerie feeling, trending in artstation.'" width="100%">
