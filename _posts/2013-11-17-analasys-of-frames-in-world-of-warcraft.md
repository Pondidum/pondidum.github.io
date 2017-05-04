---
layout: post
title: Analysis of Frames in World of Warcraft
tags: code

---

In this post we will be looking at how the `Frame` and associated objects are (probably) constructed behind the scenes.  This will all be done via inspection in lua from the games scripting engine.

The basic display item in Warcraft is the `Frame`.  Frames are not only use for displaying data, but used to listen to events in the background.  Another interesting characteristic of a `Frame` is that you cannot destroy them.  Once they are created, Frames exist for the lifetime of the UI (until the player logs out, or reloads their UI.)

First Question
---

A lot of frames are created in the lifetime of the Warcraft UI, so they should be reasonably light weight.  It seems unlikely that all methods are defined on a frame directly, as this would increase a blank/empty frames memory footprint considerably.

Because of this, it would be reasonable to expect that all methods are defined on a metatable, and that a frame created by `CreateFrame` is just a blank table with a metatable containing all the methods.  A simple implementation of `CreateFrame` could be:

```lua
function CreateFrame(type, name, parent, inherits)

	local widget = {}
	local meta = _metas[type]

	setmetatable(widget, { __index = meta })

	widget:SetName(name)
	widget:SetParent(parent)

	--something to handle templates...

	return widget

end
```

The following function will display all of the methods found on the Frame's metatable:

### Code:

```lua
local frame = CreateFrame("Frame")
local meta = getmetatable(frame)

print("meta", meta)
print("index", meta.__index)

for name, value in pairs(meta.__index) do
	print(name, value)
end
```

### Output:

	meta table: 000000000D42E610
	index table: 000000000D42E660
	IsMovable function: 000000000D07F140
	SetAlpha function: 000000000D07E800
	SetScript function: 000000000D07E0C0
	...

Interestingly, if you run this script after reloading your UI, the hash of the meta is the same every time.

The next point to investigate is how other frame types are built up.  As widgets have a hierarchical structure (see the [Widget Hierarchy][1] at wowprogramming.com), it might be the case that the `FrameMeta` has a metatable of the methods which represent `VisibleRegion`, `Region` or `ScriptObject`. The Widget Hierarchy hints that it won't be a metatable chain, as some widgets inherit multiple other types (e.g. `Frame` inherits `VisibleRegion` and `ScriptObject`).  The following function will recurse metatables, and verify if a `Frame` and a `Button` share any metatables:

### Code:

```lua
local function printTable(t)

	local meta = getmetatable(t)

	if not meta then return end
	if not meta.__index then return end

	local index = meta.__index

	print("meta:", meta, "meta.index:" index)
	printTable(meta)

end

print("Frame:")
printTable(CreateFrame("Frame"))

print("Button:")
printTable(CreateFrame("Button"))
```

### Output:

	Frame:
	meta: table: 000000000C8F8B40 meta.index: table: 000000000C8F8B90
	Button:
	meta: table: 000000000C8F8BE0 meta.index: table: 000000000C8F8C30

The output of this indicates that each Widget type has it's own metatable, which helps give a starting point to implementing a new version.

Implementing
---

The [WowInterfakes][2] project needs to be able to create all Widgets, so using a similar method as the Warcraft implementation made sense.  As there is no inheritance between Widgets, using a Mixin style for building metatables makes most sense.  The result of building the metatables is stored, and only done once on start up.

```lua
local builder = {}

builder.init = function()

	builder.metas = {}

	local frameMeta = {}

	builder.applyUIObject(frameMeta)
	builder.applyParentedObject(frameMeta)
	builder.applyRegion(frameMeta)
	builder.applyVisibleRegion(frameMeta)
	builder.applyScriptObject(frameMeta)
	builder.applyFrame(frameMeta)

	builder.metas.frame = { __index = frameMeta }

end
```

Each `apply` method mixes in the functionality for their type.  `applyRegion` gets reused for a `Texture` as well as a `Frame` for example.

Internally, all mixed in methods write and read to a table on the `self` parameter (called `__storage`), which holds each widgets values:

```lua
builder.applyFrame = function(region)

	region.SetBackdrop = function(self, backdrop)
		self.__storage.backdrop = backdrop
	end

	region.RegisterEvent = function(self, event)
		eventRegistry.register(self, event)
	end

	region.CreateTexture = function(self, name, layer, inherits, sublevel)
		return builder.createTexture(self, name, layer, inherits, sublevel)
	end

end
```

When `createFrame` is called, we create a new table with a table in `__storage`, and apply the standard frame metatable to it.  At this point, the new table is a working `Frame`, which takes up very little in the way of resources (two tables worth of memory).  Initialisation is finished by populating a few properties (some things like frame name are not publicly accessible, so they are written to the backing `__storage` directly), and apply any templates specified.

```lua
builder.createFrame = function(type, name, parent, template)

	local frame = { __storage = {} }

	setmetatable(frame, builder.metas.frame)

	frame.__storage.name = name  --no publicly accessable SetName method()
	frame:SetParent(parent)

	if template and template ~= "" then
		templateManager.apply(template, frame)
	end

	return frame

end
```


Conclusion
---

It seems likely that the `CreateFrame` method in Warcraft is defined in C, rather than in lua somewhere, so where a widget stores it's data internally is unknown.  However for the purpose of re-implementing CreateFrame, we can use a table on each widget.  Another option would have been a single table keyed by the returned widget, and store all the data centrally rather than on the individual frames.




[1]: http://wowprogramming.com/docs/widgets_hierarchy
[2]: https://github.com/Pondidum/WowInterfakes