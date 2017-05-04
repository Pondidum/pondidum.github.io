---
layout: post
title: Encapsulation in Warcraft Addons - Inheritance
tags: design code lua warcraft
---

## Using Inheritance (sort of)

When we actually need inheritance, things get a little more complicated.  We need to use two of lua's slightly harder features to get it to work: `metatables` and `colon notation`.  A little background on these will help:

### MetaTables

All "objects" in lua are tables, and tables can something called a metatable added to them.  Metatables can have special methods on them which run under certain circumstances (called metamethods), such as keys being added.  A full list of metamethods is [available here][lua-metamethods].

The metamethod we are interested in is called called `__index`, which gets triggered when a key is not found in the table.

There are two ways of using `__index`.  The first is to assign it a function, which gets passed two arguments: `table`, and `key`.  This is useful if you want to provide a default value if a key in a table isn't found, which I use in the `spellData` example [in the previous post][blog-wow-closure].

The other way of using `__index` is to pass it another table of methods to call, like in this example:

```lua
local meta = {
	print = function()
		print("Hi from the metatable")
	end
}

local actual = {
	test = function()
		print("testing")
	end
}

--wont work:
-- actual.print()

setmetatable(actual, { __index = meta })

-- now it will!
-- actual.print()
```

By calling `setmetatable` on `actual`, we provide `actual` with all the methods on `meta`.  A table can only have one meta table though, and you might break things by overwriting it (example, don't call `setmetatable` on a Frame or ActionButton...)

### Colon Notation

All methods on a table can be called in two ways; with a colon, or with a period.  The colon can be thought of as "fill in the first parameter with the table this is being called on".  For example, these two statements are equivalent:

```lua%}
local x = string.gsub("hello world", "hello", "bye")
local x = "hello world":gsub("hello", "bye")
```

In the example above, the signature of `gsub` is something like this:

```lua
local string = {
	gsub = function(self, searchTerm, replacement)
		--self is the input string
	end,
}
```

The convention used is to call the first parameter `self`.  We can now use this colon notation with metatables to make our version of inheritance.

### Combining

```lua
local base = {
	increase = function(self)
		self.count = self.count + 1
	end,
	print = function(self)
		print("The count is " .. self.count .. ".")
	end
}

local first = {
	count = 0
}
setmetatable(first, { __index = base })

local second = {
	count = 100
}
setmetatable(second, { __index = base })

--usage
first:increase()
second:increase()

first:print()		-- prints 1
first:print()		-- prints 101
```

Due to the way the colon operator works, the `self` parameter is filled in with the table calling the method, not the table the method is defined on.  So calling `first:increase()` is the same as `base.increase(first)`

## Usage

We can now take these elements, and craft a set of classes designed for reuse.  We start off with our root object (think `System.Object` if you are from a .net world.)

```lua
local class = {

	extend = function(self, this)
		return setmetatable(this, { __index = self })
	end,

	new = function(self, ...)

		local this = setmetatable({}, { __index = self })
		this:ctor(...)

		return this

	end,

	ctor = function(self, ...)
	end,
}
```

We have two methods here, `extend` and `new`.  The `new` method is nice and straight forward - it creates a new table, assigns the meta to be `class` and calls the `ctor` method (which is the one you would want to replace in sub classes).

The `extend` method takes in a new table, and applies and sets the meta to `class`.  This is what is used to inherit and add new functionality.  

For example, in my control library, I have a base class with some common methods:

```lua
local control = class:extend({

	size = function(self, config)
		self.frame:SetSize(unpack(config))
	end,

	point = function(self, config)
		self.frame:SetPoint(unpack(config))
	end,

	parent = function(self, value)
		self.frame:SetParent(value)
	end,
})
```

And then many other classes which extend the base, cilling in the `ctor` method with how to actually create the frame:

```lua
local label = control:extend({

	ctor = function(self, name, parent)
		self.frame = CreateFrame("Frame", name, parent)
		self.label = self.frame:CreateFontString()
		self.label:SetAllPoints(self.frame)
		self.label:SetFont(fonts.normal, 12)
	end,
})

local textbox  = control:extend({

	ctor = function(self, name, parent)
		self.frame = CreateFrame("editbox", name, parent, "InputBoxTemplate")
		self.frame:SetAutoFocus(false)
		self.frame:SetFont(fonts.normal, 12)
	end,

	text = function(self, value)
		self.frame:SetText(value)
	end,
})
```

Some classes, such as the textbox provide other methods where they make sense.

### Calling Base Class Methods

If we wish to start overriding a method and then call the original method within, things start to get a lot more complicated.

```lua
local class = {
	extend = function(self, this)
		this.base = self
		return setmetatable(this, { __index = self })
	end,
}

local child = class:extend({
	method = function(self)
		self.name = "child"
	end,
})

local grandchild = child:extend({
	method = function(self)
		self.base:method()
	end
})
```

While this looks like it will work, it will cause some strange and hard to debug problems (I know it will, it took me ages to figure out.)

The problem is that when you do `self.base:method()` you are effectively doing `self.base.method(self.base)`, which means the base method is referencing the wrong table!

We can solve this, but it requires a certain level of voodoo.  First we need to change our `extend` method:

```lua
extend = function(self, this)

	this.super = function(child)

		local parent = {
			__index = function(_, methodName)
				return function(_, ...)
					self[methodName](child, ...)
				end
			end
		}

		return setmetatable({}, parent)
	end

	return setmetatable(this, { __index = self })
end
```

This took me far too long to come up with and get working.  Essentially what it does is take all calls, and replace the `self` parameter with the correct table.  

This method has some restrictions, in that you can only go 'up' one level in the class hierarchy, e.g. you cannot do `item:super():super():super()`.  In practice though, I have never needed to do this.

The entirety of my class file can be found on [my github][dark-class].

### Problems

There are two disadvantages to this method of creating objects.  The first is using a table like this, you can no longer totally hide variables as you could do in the closure version.  The other is the complexity added - especially if you wish to allow base method calling, however in balance, you only need to write the `super()` functionality once (or use mine!)

When writing addons, I use both methods of encapsulation where they fit best - as like everything else in development the answer to what to use is "it depends".

[wiki-closure]: http://en.wikipedia.org/wiki/Closure_(computer_programming)
[dark-bags-groups]: https://github.com/Pondidum/Dark.Bags/tree/master/groups
[lua-metamethods]: http://lua-users.org/wiki/MetatableEvents
[blog-wow-closure]: http://andydote.co.uk/2014/11/28/encapsulation-in-warcraft-addons-closures.html
[dark-class]: https://github.com/Pondidum/Dark/blob/abcaa319ccce1bb448a1e04f1d82b8d24578acbe/class.lua
