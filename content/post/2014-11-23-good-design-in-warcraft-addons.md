---
date: "2014-11-23T00:00:00Z"
tags: design lua warcraft
title: Good Design in Warcraft Addons/Lua
---

## Lack of Encapsulation in Addons

I first noticed a lack of good design in addon code when I started trying to tweak existing addons to be slightly different.

One of the stand out examples was a Threat Meter (you know which one I mean).  It works well, but I felt like writing my own, to make it really fit into my UI, with as little overhead as possible.  Not knowing how to even begin writing a Threat Meter, I downloaded a copy, and opened its source directory... to discover that the entire addon is one 3500+ line file, and 16 Ace.* dependencies.

When I had finished my Threat Meter, I had two files (170 lines and 130 lines), and one dependency (Dark.Core, which all my addons use).  I learnt a lot while reading the source for the original threat meter - it is very customisable, is externally skinable, and has some very good optimisations in it.  But it also has a lot of unused variables (which are named very similarly to used ones), and so much of it's code *could* be separated out, making it easier to modify by newer project members.

This set of observations goes on forever when concerning addons.  The three main problems I see are:

* Pollution of the global namespace
* All code in one file
* No separation of concerns

All of this makes it harder for new developers to pick up and learn how to maintain and write addons.  They are all fairly straight forward to solve problems, so lets address them!


## Pollution of the Global Namespace

A lot of addons you find declare many variables as global so they can access them anywhere within their addon.  For example, this is pretty standard:

```lua
MyAddonEvents = CreateFrame("Frame", "MyAddonEventFrame")

MyAddonEvents:RegisterEvent("PLAYER_ENTERING_WORLD")
MyAddonEvents:SetScript("OnEvent", MyAddonEventHandler)

MyAddonEventHandler = function(self, event, ...)

	if event == "PLAYER_ENTERING_WORLD" then
		--do something useful
	end
end
```

This is an example of poluting the global namespace, as now the entire UI has access to: `MyAddonEvents`, `MyAddonEventFrame`, `MyAddonEventHandler`.  This is very trivial to rewrite to not expose anything to the global namespace:

```lua
local events = CreateFrame("Frame")
local handler = function(self, event, ...)

	if event == "PLAYER_ENTERING_WORLD" then
		--do something useful
	end

end

events:RegisterEvent("PLAYER_ENTERING_WORLD")
events:SetScript("OnEvent", handler)
```

This version exposes nothing to the global namespace, and performs exactly the same function (you can even get rid of the `handler` variable and just pass the function directly into `SetScript`).

However, by writing your code like this, you can't access any of this from another file (either a lua file, or *shudder* a frameXml file), but using namespaces we can get around this limitation without polluting the global namespace.

## Splitting into Separate Files

So, how to access local variables in other files?  Well Warcraft addons come with a feature where all lua files are provided with two arguments: `addon` and `ns`.  The first of these is a string of the addon name, and the second is an empty table.  I almost never use the `addon` parameter, but the `ns` (or "namespace") parameter is key to everything.

You can access these two variables by writing this as the first line of your lua file:

```lua
local addon, ns = ...

print("Hello from, " .. addon)
```

By using the `ns`, we can put our own variables into it to access from other files.  For example, we have an event system in one file:

*eventSystem.lua*
```lua
local addon, ns = ...

local events = CreateFrame("Frame")
local handlers = {}

events:SetScript("OnEvent", function(self, event, ...)

	local eventHandlers = handlers[event] or {}

	for i, handler in ipairs(eventHandlers) do
		handler(event, ...)
	end

end)

ns.register = function(event, handler)

	handlers[event] = handlers[event] or {}
	table.insert(handlers[event], handler)

	events:RegisterEvent(event)

end
```

Note how the `register` function is defined on the `ns`.  This means that any other file in our addon can do this to handle an event:

*goldPrinter.lua*
```lua
local addon, ns = ...

ns.register("PLAYER_MONEY", function()

	local gold = floor(money / (COPPER_PER_SILVER * SILVER_PER_GOLD))
	local silver = floor((money - (gold * COPPER_PER_SILVER * SILVER_PER_GOLD)) / COPPER_PER_SILVER)
	local copper = mod(money, COPPER_PER_SILVER)

	local moneyString = ""
	local separator = ""

	if ( gold > 0 ) then
		moneyString = format(GOLD_AMOUNT_TEXTURE, gold, 0, 0)
		separator = " "
	end
	if ( silver > 0 ) then
		moneyString = moneyString .. separator .. format(SILVER_AMOUNT_TEXTURE, silver, 0, 0)
		separator = " "
	end
	if ( copper > 0 or moneyString == "" ) then
		moneyString = moneyString .. separator .. format(COPPER_AMOUNT_TEXTURE, copper, 0, 0)
	end

	print("You now have " .. moneyString)

end)
```

A pretty trivial example, but we have managed to write a two file addon, without putting **anything** in the global namespace.

We have also managed to separate our concerns - the `goldPrinter` does not care what raises the events, and the `eventSystem` knows nothing about gold printing, just how to delegate events.  There is also an efficiency here too - anything else in our addon that needs events uses the same eventSystem, meaning we only need to create one frame for the entire addon to receive events.

## Structure

Now that we can separate things into individual files, we gain a slightly different problem - how to organise those files.  I found over time that I end up with roughly the same structure each time, and others might benefit from it too.

All my addons start with four files:

* AddonName.toc
* initialise.lua
* config.lua
* run.lua

The toc file, other than the usual header information is laid out in the order the files will run, for example this is the file segment of my bags addon's toc file:

```powershell
initialise.lua
config.lua

models\classifier.lua
models\classifiers\equipmentSet.lua
models\itemModel.lua
models\model.lua

groups\group.lua
groups\bagGroup.lua
groups\bagContainer.lua

views\item.lua
views\goldDisplay.lua
views\currencyDisplay.lua
views\bankBagBar.lua

sets\container.lua
sets\bag.lua
sets\bank.lua

run.lua
```

The `initialise` lua file is the first thing to run.  All this tends to do is setup any sub-namespaces on `ns`, and copy in external dependencies to `ns.lib`:

```lua
local addon, ns = ...

ns.models = {}
ns.groups = {}
ns.views = {}
ns.sets = {}

local core = Dark.core

ns.lib = {
	fonts = core.fonts,
	events = core.events,
	slash = core.slash,
}
```
By copying in the dependencies, we not only save a global lookup each time we need say the event system, but we also have an abstraction point.  If we want to replace the event system, as long as the replacement has the right function names, we can just assign the new one to the lib: `ns.lib.events = replacementEvents:new()`

The sub namespaces correspond to folders on in the addon (much the same practice used by c# developers), so for example the `classifier.lua` file might have this in it:

```lua
local addon, ns = ...

local classifier = {
	new = function() end,
	update = function() end,
	classify = function(item) end,
}

ns.models.classifier = classifier
```

The config file should be fairly simple, with not much more than a couple of tables in it:
```lua
local addon, ns = ...

ns.config = {
	buttonSize = 24,
	spacing = 4,
	screenPadding = 10,
	currencies = {
		823, -- apexis
		824,  -- garrison resources
	}
}
```

And finally, the `run.lua` file is what makes your addon come to life:

```lua
local addon, ns = ...

local sets = ns.sets

local pack = sets.bag:new()
local bank = sets.bank:new()

local ui = ns.controllers.uiIntegration.new(pack.frame, bank.frame)
ui.hook()

--expose
DarkBags = {
	addClassifier = ns.classifiers.add
}
```

If you need to expose something to the entire UI or other addons, that's fine.  But make sure you only expose what you want to.  In the example above the `DarkBags` global only has one method - `addClassifier`, because that is all I want other addons to be able to do.

## Wrapping Up

I hope this helps other people with their addons - I know I wish that I had gotten to this structure and style a lot sooner than I did.

There will be a few more posts incoming covering encapsulation, objects and inheritance in more detail, so stay tuned.
